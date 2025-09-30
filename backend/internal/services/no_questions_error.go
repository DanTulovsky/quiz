package services

import (
	"fmt"

	contextutils "quizapp/internal/utils"
)

// NoQuestionsAvailableError is returned when no suitable questions can be found for assignment.
type NoQuestionsAvailableError struct {
	Language       string
	Level          string
	CandidateIDs   []int
	CandidateCount int
	TotalMatching  int
}

func (e *NoQuestionsAvailableError) Error() string {
	return fmt.Sprintf("no questions available for assignment (language=%s level=%s candidate_count=%d total_matching=%d)", e.Language, e.Level, e.CandidateCount, e.TotalMatching)
}

// Unwrap allows errors.Is(..., contextutils.ErrNoQuestionsAvailable) to work.
func (e *NoQuestionsAvailableError) Unwrap() error {
	return contextutils.ErrNoQuestionsAvailable
}
