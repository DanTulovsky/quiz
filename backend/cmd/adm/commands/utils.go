package commands

import (
	"database/sql"
	"fmt"
	"strings"
)

// maskDatabaseURL masks sensitive parts of the database URL for display
func maskDatabaseURL(url string) string {
	// Simple masking for display purposes
	if strings.Contains(url, "@") {
		parts := strings.Split(url, "@")
		if len(parts) == 2 {
			return "postgres://***:***@" + parts[1]
		}
	}
	return url
}

// getDatabaseInfo returns database connection information
func getDatabaseInfo(db *sql.DB) string {
	if db == nil {
		return "Not connected"
	}

	// Try to get database name
	var dbName string
	err := db.QueryRow("SELECT current_database()").Scan(&dbName)
	if err != nil {
		return "Connected (unknown database)"
	}

	// Try to get host information
	var host string
	err = db.QueryRow("SELECT inet_server_addr()::text").Scan(&host)
	if err != nil {
		return fmt.Sprintf("Connected to %s", dbName)
	}

	return fmt.Sprintf("Connected to %s on %s", dbName, host)
}
