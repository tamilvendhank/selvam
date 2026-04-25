package common

import "errors"

var (
	ErrInvalidServiceRequest       = errors.New("invalid service request")
	ErrNothingToSubmit             = errors.New("nothing to submit")
	ErrNothingToPoll               = errors.New("nothing to poll")
	ErrNothingToReconcile          = errors.New("nothing to reconcile")
	ErrNothingToValidate           = errors.New("nothing to validate")
	ErrNothingToMaterialize        = errors.New("nothing to materialize")
	ErrNothingToFinalize           = errors.New("nothing to finalize")
	ErrWorkflowNotReadyToContinue  = errors.New("workflow is not ready to continue")
	ErrNoEligibleCandidates        = errors.New("no eligible candidates")
	ErrWorkerLeaseUnavailable      = errors.New("worker lease unavailable")
	ErrWorkerLeaseConflict         = errors.New("worker lease conflict")
	ErrUnsupportedServiceOperation = errors.New("unsupported service operation")
)
