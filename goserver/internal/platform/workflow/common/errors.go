package common

import "errors"

var (
	ErrWorkflowNotFound         = errors.New("workflow run not found")
	ErrWorkflowNotResumable     = errors.New("workflow run is not resumable")
	ErrWorkflowNotReconciliable = errors.New("workflow run is not reconcilable")
	ErrWorkflowAlreadyTerminal  = errors.New("workflow run is already terminal")
	ErrWorkflowWaitingExternal  = errors.New("workflow run is waiting on external dependencies")
	ErrInvalidWorkflowRequest   = errors.New("invalid workflow request")
	ErrStepPreconditionFailed   = errors.New("workflow step precondition failed")
	ErrContinuationNotReady     = errors.New("workflow continuation is not ready")
)
