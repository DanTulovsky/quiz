package worker

import (
	"context"
	"testing"

	"quizapp/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestWorker_GetEligibleQuestionCount_Error(t *testing.T) {
	mockQuestionService := &mockQuestionService{}
	mockLearningService := &mockLearningService{}
	mockWorkerService := &mockWorkerService{}

	w := &Worker{
		questionService: mockQuestionService,
		learningService: mockLearningService,
		workerService:   mockWorkerService,
	}

	ctx := context.Background()

	count, err := w.getEligibleQuestionCount(ctx, 1, "italian", "A1", models.Vocabulary)
	assert.Error(t, err)
	assert.Equal(t, 0, count)
}
