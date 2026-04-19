package shared

import "strings"

const DefaultModel = "gpt-4.1-mini"

type ReasoningProfile struct {
	Supported    bool
	Options      []string
	DefaultValue string
}

func NormalizeModelName(model string) string {
	trimmed := strings.TrimSpace(model)
	if trimmed == "" {
		return DefaultModel
	}

	return trimmed
}

func GetReasoningProfile(model string) ReasoningProfile {
	normalized := normalizeLower(model)

	if normalized == "" {
		return ReasoningProfile{
			Supported:    true,
			Options:      []string{"none", "minimal", "low", "medium", "high", "xhigh"},
			DefaultValue: "medium",
		}
	}

	if strings.Contains(normalized, "pro") {
		return ReasoningProfile{
			Supported:    true,
			Options:      []string{"high"},
			DefaultValue: "high",
		}
	}

	if strings.HasPrefix(normalized, "gpt-5.1") {
		return ReasoningProfile{
			Supported:    true,
			Options:      []string{"none", "low", "medium", "high"},
			DefaultValue: "none",
		}
	}

	if strings.HasPrefix(normalized, "gpt-5.2") ||
		strings.HasPrefix(normalized, "gpt-5.3") ||
		strings.HasPrefix(normalized, "gpt-5.4") ||
		strings.HasPrefix(normalized, "gpt-5.5") ||
		strings.HasPrefix(normalized, "gpt-6") {
		return ReasoningProfile{
			Supported:    true,
			Options:      []string{"none", "minimal", "low", "medium", "high", "xhigh"},
			DefaultValue: "medium",
		}
	}

	if strings.HasPrefix(normalized, "gpt-5") || strings.HasPrefix(normalized, "o") {
		return ReasoningProfile{
			Supported:    true,
			Options:      []string{"minimal", "low", "medium", "high"},
			DefaultValue: "medium",
		}
	}

	return ReasoningProfile{}
}

func NormalizeReasoningEffort(model, reasoningEffort string) *string {
	profile := GetReasoningProfile(model)
	if !profile.Supported {
		return nil
	}

	normalized := normalizeLower(reasoningEffort)
	for _, option := range profile.Options {
		if normalized == option {
			return StringPtr(option)
		}
	}

	return StringPtr(profile.DefaultValue)
}

func normalizeLower(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
