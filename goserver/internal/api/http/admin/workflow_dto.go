package admin

import (
	"time"

	domaincommon "goserver/internal/domain/common"
	continuationsvc "goserver/internal/service/continuation"
)

type WorkflowRunListItemDTO struct {
	WorkflowRunID        string                         `json:"workflowRunId"`
	BookType             domaincommon.BookType          `json:"bookType"`
	RunType              domaincommon.WorkflowRunType   `json:"runType"`
	Status               domaincommon.WorkflowRunStatus `json:"status"`
	StartedAt            time.Time                      `json:"startedAt"`
	CompletedAt          *time.Time                     `json:"completedAt,omitempty"`
	CompaniesScanned     int                            `json:"companiesScannedCount,omitempty"`
	ReviewsCreated       int                            `json:"reviewsCreatedCount,omitempty"`
	ErrorsCount          int                            `json:"errorsCount,omitempty"`
	ConfigSnapshotID     string                         `json:"configSnapshotId,omitempty"`
	Notes                string                         `json:"notes,omitempty"`
	CreatedAt            time.Time                      `json:"createdAt"`
	UpdatedAt            time.Time                      `json:"updatedAt"`
}

type WorkflowRunDetailDTO struct {
	WorkflowRunListItemDTO
	StepSummary        StatusCountsDTO                         `json:"stepSummary,omitempty"`
	AIBatchJobSummary  StatusCountsDTO                         `json:"aiBatchJobSummary,omitempty"`
	AIBatchItemSummary StatusCountsDTO                         `json:"aiBatchItemSummary,omitempty"`
	ValidationSummary  ValidationCountsDTO                     `json:"validationSummary,omitempty"`
	CurrentContinuation *WorkflowContinuationDecisionDTO        `json:"currentContinuation,omitempty"`
	StepRefs           []WorkflowStepDTO                       `json:"stepRefs,omitempty"`
	Warnings           []string                                `json:"warnings,omitempty"`
}

type WorkflowStepDTO struct {
	WorkflowStepRunID string                          `json:"workflowStepRunId,omitempty"`
	WorkflowRunID     string                          `json:"workflowRunId,omitempty"`
	StepName          domaincommon.WorkflowStepName   `json:"stepName"`
	Status            domaincommon.WorkflowStepStatus `json:"status"`
	StartedAt         *time.Time                      `json:"startedAt,omitempty"`
	CompletedAt       *time.Time                      `json:"completedAt,omitempty"`
	ErrorSummary      string                          `json:"errorSummary,omitempty"`
	Metadata          map[string]any                  `json:"metadata,omitempty"`
	CreatedAt         time.Time                       `json:"createdAt"`
	UpdatedAt         time.Time                       `json:"updatedAt"`
}

type WorkflowStatusDTO struct {
	WorkflowRunID       string                         `json:"workflowRunId"`
	BookType            domaincommon.BookType          `json:"bookType,omitempty"`
	RunType             domaincommon.WorkflowRunType   `json:"runType,omitempty"`
	Status              domaincommon.WorkflowRunStatus `json:"status"`
	CurrentStep         domaincommon.WorkflowStepName  `json:"currentStep,omitempty"`
	NextLikelyStep      domaincommon.WorkflowStepName  `json:"nextLikelyStep,omitempty"`
	WaitingExternal     bool                           `json:"waitingExternal"`
	Blocked             bool                           `json:"blocked"`
	Terminal            bool                           `json:"terminal"`
	StepSummary         StatusCountsDTO                `json:"stepSummary,omitempty"`
	AIBatchJobSummary   StatusCountsDTO                `json:"aiBatchJobSummary,omitempty"`
	AIBatchItemSummary  StatusCountsDTO                `json:"aiBatchItemSummary,omitempty"`
	ValidationSummary   ValidationCountsDTO            `json:"validationSummary,omitempty"`
	LatestErrors        []string                       `json:"latestErrors,omitempty"`
	Continuation        *WorkflowContinuationDecisionDTO `json:"continuation,omitempty"`
	UpdatedAt           time.Time                      `json:"updatedAt"`
}

type WorkflowSummaryDTO struct {
	WorkflowRun          WorkflowRunListItemDTO                 `json:"workflowRun"`
	StepProgress         StatusCountsDTO                         `json:"stepProgress"`
	BatchJobSummary      StatusCountsDTO                         `json:"batchJobSummary"`
	BatchItemSummary     StatusCountsDTO                         `json:"batchItemSummary"`
	ValidationSummary    ValidationCountsDTO                     `json:"validationSummary"`
	MaterializationSummary map[string]int                        `json:"materializationSummary,omitempty"`
	ContinuationSummary  *WorkflowContinuationDecisionDTO        `json:"continuationSummary,omitempty"`
	NextSuggestedAction  string                                  `json:"nextSuggestedAction,omitempty"`
	Warnings             []string                                `json:"warnings,omitempty"`
}

type WorkflowContinuationDecisionDTO struct {
	WorkflowRunID            string                                           `json:"workflowRunId"`
	BookType                 domaincommon.BookType                            `json:"bookType,omitempty"`
	CurrentStatus            domaincommon.WorkflowRunStatus                   `json:"currentStatus,omitempty"`
	Readiness                continuationsvc.WorkflowContinuationReadiness    `json:"readiness,omitempty"`
	ReadyToContinue          bool                                             `json:"readyToContinue"`
	WaitingOnExternalJobs    bool                                             `json:"waitingOnExternalJobs,omitempty"`
	WaitingOnValidation      bool                                             `json:"waitingOnValidation,omitempty"`
	WaitingOnMaterialization bool                                             `json:"waitingOnMaterialization,omitempty"`
	WaitingOnFinalization    bool                                             `json:"waitingOnFinalization,omitempty"`
	NextSuggestedStep        domaincommon.WorkflowStepName                    `json:"nextSuggestedStep,omitempty"`
	ContinuationReason       continuationsvc.ContinuationReason               `json:"continuationReason,omitempty"`
	Blockers                 []any                                            `json:"blockers,omitempty"`
	Counts                   continuationsvc.WorkflowContinuationCounts       `json:"counts,omitempty"`
	Summary                  any                                              `json:"summary,omitempty"`
}
