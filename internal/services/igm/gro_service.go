package igm

import (
	"context"

	"uois-gateway/internal/models"

	"go.uber.org/zap"
)

// GROService provides GRO (Grievance Redressal Officer) details
type GROService struct {
	logger *zap.Logger
}

// NewGROService creates a new GRO service
func NewGROService(logger *zap.Logger) *GROService {
	return &GROService{
		logger: logger,
	}
}

// GetGRODetails returns GRO details for a given issue type
func (s *GROService) GetGRODetails(ctx context.Context, issueType models.IssueType) (*models.GRO, error) {
	level := models.GetGROLevelForIssueType(issueType)
	return s.getDefaultGRO(level), nil
}

func (s *GROService) getDefaultGRO(level string) *models.GRO {
	switch level {
	case "L1":
		return &models.GRO{
			Level:       "L1",
			Name:        "L1 Support Officer",
			Email:       "l1-support@example.com",
			Phone:       "+91-1234567890",
			ContactType: "PRIMARY",
		}
	case "L2":
		return &models.GRO{
			Level:       "L2",
			Name:        "L2 Support Manager",
			Email:       "l2-support@example.com",
			Phone:       "+91-1234567891",
			ContactType: "PRIMARY",
		}
	case "L3":
		return &models.GRO{
			Level:       "L3",
			Name:        "L3 Escalation Manager",
			Email:       "l3-support@example.com",
			Phone:       "+91-1234567892",
			ContactType: "PRIMARY",
		}
	default:
		return &models.GRO{
			Level:       "L1",
			Name:        "Default Support Officer",
			Email:       "support@example.com",
			Phone:       "+91-1234567890",
			ContactType: "PRIMARY",
		}
	}
}
