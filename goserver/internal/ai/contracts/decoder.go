package contracts

import (
	"encoding/json"
	"fmt"
)

func DecodeInvestingReviewInput(data []byte) (*InvestingReviewInputEnvelope, error) {
	var envelope InvestingReviewInputEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("decode investing review input envelope: %w", err)
	}
	if err := ValidateInvestingReviewInputEnvelope(&envelope); err != nil {
		return nil, err
	}
	return &envelope, nil
}

func DecodeInvestingReviewOutput(data []byte) (*InvestingReviewOutputEnvelope, error) {
	var envelope InvestingReviewOutputEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("decode investing review output envelope: %w", err)
	}
	if err := ValidateInvestingReviewOutputEnvelope(&envelope); err != nil {
		return nil, err
	}
	return &envelope, nil
}
