// Package main contains a small backfill tool that finds daily_question_assignments
// that are missing a corresponding daily_assignment_responses entry and attempts
// to map them to the latest user_responses within the user's local day.
package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	var dbURL string
	var batchSize int
	var dryRun bool
	var maxRows int

	flag.StringVar(&dbURL, "db", os.Getenv("DATABASE_URL"), "Postgres connection string (or set DATABASE_URL)")
	flag.IntVar(&batchSize, "batch", 500, "Number of assignments to process per batch")
	flag.BoolVar(&dryRun, "dry-run", true, "If true, don't write mappings; just print what would be written")
	flag.IntVar(&maxRows, "max", 0, "Maximum number of assignments to process (0 = no limit)")
	flag.Parse()

	if dbURL == "" {
		log.Fatal("database URL must be provided via -db or DATABASE_URL")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			log.Fatalf("failed to close db: %v", cerr)
		}
	}()

	ctx := context.Background()

	processed := 0
	lastID := 0
	for {
		if maxRows > 0 && processed >= maxRows {
			log.Printf("reached max %d rows, stopping", maxRows)
			break
		}

		// select assignments without mapping, only ids greater than lastID to avoid reprocessing
		rows, err := db.QueryContext(ctx, `
            SELECT dqa.id, dqa.user_id, dqa.question_id, dqa.assignment_date
            FROM daily_question_assignments dqa
            LEFT JOIN daily_assignment_responses dar ON dar.assignment_id = dqa.id
            WHERE dar.id IS NULL AND dqa.id > $2
            ORDER BY dqa.id
            LIMIT $1
        `, batchSize, lastID)
		if err != nil {
			log.Fatalf("failed to query assignments: %v", err)
		}

		var assignments []struct {
			ID             int
			UserID         int
			QuestionID     int
			AssignmentDate time.Time
		}

		for rows.Next() {
			var a struct {
				ID             int
				UserID         int
				QuestionID     int
				AssignmentDate time.Time
			}
			if err := rows.Scan(&a.ID, &a.UserID, &a.QuestionID, &a.AssignmentDate); err != nil {
				if cerr := rows.Close(); cerr != nil {
					log.Fatalf("scan assignment: %v; also failed to close rows: %v", err, cerr)
				}
				log.Fatalf("scan assignment: %v", err)
			}
			assignments = append(assignments, a)
		}
		if cerr := rows.Close(); cerr != nil {
			log.Printf("warning: failed to close rows: %v", cerr)
		}

		if len(assignments) == 0 {
			log.Println("no more assignments to process; done")
			break
		}

		// advance lastID so we don't refetch the same unmapped assignments
		lastID = assignments[len(assignments)-1].ID

		for _, a := range assignments {
			if maxRows > 0 && processed >= maxRows {
				break
			}

			// get user's timezone
			var tz sql.NullString
			err := db.QueryRowContext(ctx, `SELECT timezone FROM users WHERE id = $1`, a.UserID).Scan(&tz)
			loc := time.UTC
			if err == nil && tz.Valid && tz.String != "" {
				if l, err2 := time.LoadLocation(tz.String); err2 == nil {
					loc = l
				} else {
					log.Printf("warning: failed to load location '%s' for user %d: %v; defaulting to UTC", tz.String, a.UserID, err2)
				}
			}

			// construct local-day range for assignment_date in user's timezone
			year, month, day := a.AssignmentDate.Date()
			localStart := time.Date(year, month, day, 0, 0, 0, 0, loc)
			localEnd := localStart.Add(24 * time.Hour)
			// convert to UTC for DB comparison (created_at stored as timestamptz)
			startUTC := localStart.UTC()
			endUTC := localEnd.UTC()

			// find latest response in that local day
			var respID int
			var respCreated time.Time
			err = db.QueryRowContext(ctx, `
                SELECT id, created_at FROM user_responses
                WHERE user_id = $1 AND question_id = $2 AND created_at >= $3 AND created_at < $4
                ORDER BY created_at DESC LIMIT 1
            `, a.UserID, a.QuestionID, startUTC, endUTC).Scan(&respID, &respCreated)
			if err == sql.ErrNoRows {
				log.Printf("no response for assignment id=%d user=%d question=%d on local date %s (tz=%s)", a.ID, a.UserID, a.QuestionID, a.AssignmentDate.Format("2006-01-02"), loc.String())
				processed++
				continue
			}
			if err != nil {
				log.Fatalf("query response for assignment %d: %v", a.ID, err)
			}

			if dryRun {
				log.Printf("[dry-run] would insert mapping: assignment=%d -> user_response=%d (created_at=%s) tz=%s", a.ID, respID, respCreated.Format(time.RFC3339), loc.String())
			} else {
				_, err = db.ExecContext(ctx, `
                    INSERT INTO daily_assignment_responses (assignment_id, user_response_id, created_at)
                    VALUES ($1, $2, $3)
                    ON CONFLICT (assignment_id) DO UPDATE SET user_response_id = EXCLUDED.user_response_id, created_at = EXCLUDED.created_at
                `, a.ID, respID, respCreated)
				if err != nil {
					log.Fatalf("failed to insert mapping for assignment %d -> response %d: %v", a.ID, respID, err)
				}
				log.Printf("inserted mapping: assignment=%d -> user_response=%d", a.ID, respID)
			}

			processed++
		}

		// small pause to avoid overwhelming DB
		time.Sleep(200 * time.Millisecond)
	}

	log.Printf("done; processed %d assignments", processed)
}
