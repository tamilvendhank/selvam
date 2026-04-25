package common

import (
	"context"
	"time"
)

type ClockPort interface {
	Now() time.Time
}

type IDGeneratorPort interface {
	NewID() string
}

// ConfigAccessPort gives service implementations a small way to read typed
// configuration snapshots without coupling this contract layer to a concrete
// config package.
type ConfigAccessPort interface {
	Get(ctx context.Context, key string) (any, error)
}
