package projection

import "context"

type ProjectionUpdateService interface {
	UpdateProjections(ctx context.Context, request UpdateProjectionsRequest) (*UpdateProjectionsResult, error)
}
