package projection

import (
	domaincommon "goserver/internal/domain/common"
	servicecommon "goserver/internal/service/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProjectionTarget string

const (
	ProjectionTargetCompanyState ProjectionTarget = "company_state"
	ProjectionTargetPosition     ProjectionTarget = "position"
	ProjectionTargetReview       ProjectionTarget = "review"
	ProjectionTargetWorkflow     ProjectionTarget = "workflow"
)

type UpdateProjectionsRequest struct {
	WorkflowRunID primitive.ObjectID    `json:"workflowRunId,omitempty"`
	ReviewIDs     []primitive.ObjectID  `json:"reviewIds,omitempty"`
	CompanyIDs    []primitive.ObjectID  `json:"companyIds,omitempty"`
	BookType      domaincommon.BookType `json:"bookType,omitempty"`
	Targets       []ProjectionTarget    `json:"targets,omitempty"`
	DryRun        bool                  `json:"dryRun,omitempty"`
	Force         bool                  `json:"force,omitempty"`
	InitiatedBy   string                `json:"initiatedBy,omitempty"`
	CorrelationID string                `json:"correlationId,omitempty"`
}

func (request UpdateProjectionsRequest) Validate() error {
	if request.WorkflowRunID.IsZero() && len(request.ReviewIDs) == 0 && len(request.CompanyIDs) == 0 {
		return servicecommon.ValidateAtLeastOneObjectID(map[string]primitive.ObjectID{
			"workflowRunId": request.WorkflowRunID,
		})
	}
	for _, reviewID := range request.ReviewIDs {
		if reviewID.IsZero() {
			return servicecommon.ValidateRequiredObjectID("reviewIds", reviewID)
		}
	}
	for _, companyID := range request.CompanyIDs {
		if companyID.IsZero() {
			return servicecommon.ValidateRequiredObjectID("companyIds", companyID)
		}
	}
	if err := servicecommon.ValidateOptionalBookType(request.BookType); err != nil {
		return err
	}
	if err := servicecommon.ValidateOptionalText("initiatedBy", request.InitiatedBy); err != nil {
		return err
	}
	return servicecommon.ValidateOptionalText("correlationId", request.CorrelationID)
}

type ProjectionUpdateRef struct {
	Target        ProjectionTarget   `json:"target"`
	ID            primitive.ObjectID `json:"id,omitempty"`
	CompanyID     primitive.ObjectID `json:"companyId,omitempty"`
	ReviewID      primitive.ObjectID `json:"reviewId,omitempty"`
	WorkflowRunID primitive.ObjectID `json:"workflowRunId,omitempty"`
	Updated       bool               `json:"updated,omitempty"`
}

type UpdateProjectionsResult struct {
	WorkflowRunID   primitive.ObjectID                    `json:"workflowRunId,omitempty"`
	UpdatedRefs     []ProjectionUpdateRef                 `json:"updatedRefs,omitempty"`
	SkippedRefs     []ProjectionUpdateRef                 `json:"skippedRefs,omitempty"`
	PartialFailures []servicecommon.PartialFailure        `json:"partialFailures,omitempty"`
	Summary         servicecommon.ProjectionUpdateSummary `json:"summary,omitempty"`
}

func (result UpdateProjectionsResult) HasFailures() bool {
	return len(result.PartialFailures) > 0 || result.Summary.HasFailures()
}
