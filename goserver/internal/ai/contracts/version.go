package contracts

const (
	AIReviewInputSchemaVersion          = "ai-review-input.v1"
	AIReviewOutputEnvelopeSchemaVersion = "ai-review-output-envelope.v1"
	InvestingReviewOutputSchemaVersion  = "investing-review-output.v1"
	DefaultInvestingReviewPromptVersion = "investing-review-v1"

	// JSONSchemaDraft202012 is used because it is widely supported and expressive
	// enough for enum, range, and object contract checks without provider coupling.
	JSONSchemaDraft202012 = "https://json-schema.org/draft/2020-12/schema"
)

type ContractVersionMetadata struct {
	ContractName        string `json:"contract_name"`
	SchemaVersion       string `json:"schema_version"`
	PromptVersion       string `json:"prompt_version,omitempty"`
	OutputSchemaVersion string `json:"output_schema_version,omitempty"`
}

func InvestingReviewContractVersionMetadata() ContractVersionMetadata {
	return ContractVersionMetadata{
		ContractName:        "investing_company_review",
		SchemaVersion:       AIReviewInputSchemaVersion,
		PromptVersion:       DefaultInvestingReviewPromptVersion,
		OutputSchemaVersion: InvestingReviewOutputSchemaVersion,
	}
}
