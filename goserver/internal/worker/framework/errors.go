package framework

import "errors"

var (
	ErrInvalidWorkerOptions = errors.New("worker framework: invalid options")
	ErrWorkerNil            = errors.New("worker framework: worker is nil")
	ErrWorkerTimeout        = errors.New("worker framework: worker iteration timed out")
	ErrWorkerStopped        = errors.New("worker framework: worker stopped")
)
