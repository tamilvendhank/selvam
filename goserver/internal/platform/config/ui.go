package config

type UIConfig struct {
	DefaultPageSize                 int    `json:"defaultPageSize" yaml:"defaultPageSize"`
	MaxPageSize                     int    `json:"maxPageSize" yaml:"maxPageSize"`
	DefaultSortField                string `json:"defaultSortField,omitempty" yaml:"defaultSortField,omitempty"`
	EnableAdminRetryControls        bool   `json:"enableAdminRetryControls" yaml:"enableAdminRetryControls"`
	EnableRawAIResultInspection     bool   `json:"enableRawAIResultInspection" yaml:"enableRawAIResultInspection"`
	EnableValidationErrorInspection bool   `json:"enableValidationErrorInspection" yaml:"enableValidationErrorInspection"`
}
