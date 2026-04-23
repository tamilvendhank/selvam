package ports

import "goserver/internal/platform/domain"

type StartInvestingWorkflowRequest struct {
	RunType         domain.WorkflowRunType `json:"runType"`
	Mode            domain.InvestingMode   `json:"mode"`
	CompanyIDs      []string               `json:"companyIds,omitempty"`
	Limit           int                    `json:"limit,omitempty"`
	ReplayFromRunID string                 `json:"replayFromRunId,omitempty"`
	IdempotencyKey  string                 `json:"idempotencyKey,omitempty"`
	DryRun          bool                   `json:"dryRun"`
	Notes           string                 `json:"notes,omitempty"`
	RequestedBy     string                 `json:"requestedBy,omitempty"`
}

type StartTradingWorkflowRequest struct {
	RunType        domain.WorkflowRunType `json:"runType"`
	IdempotencyKey string                 `json:"idempotencyKey,omitempty"`
	DryRun         bool                   `json:"dryRun"`
	Notes          string                 `json:"notes,omitempty"`
	RequestedBy    string                 `json:"requestedBy,omitempty"`
}
