package igm

import (
	"context"
	"testing"

	"uois-gateway/internal/models"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestGROService_GetGRODetails_ForIssue(t *testing.T) {
	logger := zap.NewNop()
	service := NewGROService(logger)

	gro, err := service.GetGRODetails(context.Background(), models.IssueTypeIssue)
	assert.NoError(t, err)
	assert.NotNil(t, gro)
	assert.Equal(t, "L1", gro.Level)
	assert.NotEmpty(t, gro.Name)
	assert.NotEmpty(t, gro.Email)
}

func TestGROService_GetGRODetails_ForGrievance(t *testing.T) {
	logger := zap.NewNop()
	service := NewGROService(logger)

	gro, err := service.GetGRODetails(context.Background(), models.IssueTypeGrievance)
	assert.NoError(t, err)
	assert.NotNil(t, gro)
	assert.Equal(t, "L2", gro.Level)
}

func TestGROService_GetGRODetails_ForDispute(t *testing.T) {
	logger := zap.NewNop()
	service := NewGROService(logger)

	gro, err := service.GetGRODetails(context.Background(), models.IssueTypeDispute)
	assert.NoError(t, err)
	assert.NotNil(t, gro)
	assert.Equal(t, "L3", gro.Level)
}
