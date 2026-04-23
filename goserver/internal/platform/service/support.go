package service

import (
	"time"

	"goserver/internal/platform/ports"
)

type systemTimeProvider struct{}

func (systemTimeProvider) Now() time.Time {
	return time.Now().UTC()
}

func resolveTimeProvider(provider ports.TimeProvider) ports.TimeProvider {
	if provider != nil {
		return provider
	}

	return systemTimeProvider{}
}

func ResolveTimeProviderForWorkflow(provider ports.TimeProvider) ports.TimeProvider {
	return resolveTimeProvider(provider)
}
