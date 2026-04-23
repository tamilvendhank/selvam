package service

import "errors"

var (
	ErrNotFound          = errors.New("not found")
	ErrImmutableReview   = errors.New("review is immutable")
	ErrInvalidTransition = errors.New("invalid workflow step transition")
)
