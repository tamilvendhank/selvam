package allocation

import "context"

type CapitalCandidateBuilderService interface {
	BuildCapitalCandidates(ctx context.Context, request BuildCapitalCandidatesRequest) (*BuildCapitalCandidatesResult, error)
}

type CapitalAllocationService interface {
	AllocateCapital(ctx context.Context, request AllocateCapitalRequest) (*AllocateCapitalResult, error)
}
