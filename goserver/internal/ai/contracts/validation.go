package contracts

import (
	"fmt"
	"math"
	"strings"

	"goserver/internal/domain/common"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const compareAgainstOwnHistory = "own_history"

func ValidateInvestingReviewInputEnvelope(envelope *InvestingReviewInputEnvelope) error {
	if envelope == nil {
		return fmt.Errorf("investing review input envelope is required")
	}
	if err := validateInputEnvelopeHeader(*envelope); err != nil {
		return err
	}
	if err := validateInvestingReviewInputPayload(envelope.Payload); err != nil {
		return err
	}
	if envelope.Payload.Company.CompanyID != "" && envelope.Payload.Company.CompanyID != envelope.CompanyID {
		return fmt.Errorf("payload.company.company_id must match envelope company_id")
	}
	if envelope.Payload.Company.Symbol != "" && !strings.EqualFold(envelope.Payload.Company.Symbol, envelope.Symbol) {
		return fmt.Errorf("payload.company.symbol must match envelope symbol")
	}
	return nil
}

func ValidateInvestingReviewOutputEnvelope(envelope *InvestingReviewOutputEnvelope) error {
	if envelope == nil {
		return fmt.Errorf("investing review output envelope is required")
	}
	if err := validateOutputEnvelopeHeader(*envelope); err != nil {
		return err
	}
	if err := ValidateInvestingReviewOutputPayload(envelope.Payload); err != nil {
		return err
	}
	if envelope.Payload.CompanyID != envelope.CompanyID {
		return fmt.Errorf("payload.company_id must match envelope company_id")
	}
	if !strings.EqualFold(envelope.Payload.Symbol, envelope.Symbol) {
		return fmt.Errorf("payload.symbol must match envelope symbol")
	}
	return nil
}

func ValidateInvestingReviewOutputPayload(payload InvestingReviewOutputPayload) error {
	if err := validateRequiredObjectID("payload.company_id", payload.CompanyID); err != nil {
		return err
	}
	if err := common.RequireString("payload.symbol", payload.Symbol); err != nil {
		return err
	}
	if err := common.RequireTime("payload.review_date", payload.ReviewDate); err != nil {
		return err
	}
	if !payload.Mode.IsValid() {
		return fmt.Errorf("invalid payload.mode %q", payload.Mode)
	}
	if err := common.ValidateComputedScore("payload.weighted_total_score", payload.WeightedTotalScore); err != nil {
		return err
	}
	if err := common.ValidateUnitInterval("payload.confidence_score", payload.ConfidenceScore); err != nil {
		return err
	}
	if payload.HardGateFailed && len(payload.HardGateFailureReasons) == 0 {
		return fmt.Errorf("payload.hard_gate_failure_reasons is required when hard_gate_failed is true")
	}
	if err := common.ValidateStringSlice("payload.hard_gate_failure_reasons", payload.HardGateFailureReasons); err != nil {
		return err
	}
	if !payload.SuggestedAction.IsValid() {
		return fmt.Errorf("invalid payload.suggested_action %q", payload.SuggestedAction)
	}
	if !payload.SuggestedBucket.IsValid() {
		return fmt.Errorf("invalid payload.suggested_bucket %q", payload.SuggestedBucket)
	}
	if err := common.RequireString("payload.action_rationale_summary", payload.ActionRationaleSummary); err != nil {
		return err
	}
	if payload.CapitalPriorityScore != nil {
		if err := common.ValidateScore("payload.capital_priority_score", *payload.CapitalPriorityScore); err != nil {
			return err
		}
	}
	if payload.RecommendedPositionTargetPct != nil {
		if err := common.ValidatePercentage("payload.recommended_position_target_pct", *payload.RecommendedPositionTargetPct); err != nil {
			return err
		}
	}
	if payload.RecommendedPositionCapPct != nil {
		if err := common.ValidatePercentage("payload.recommended_position_cap_pct", *payload.RecommendedPositionCapPct); err != nil {
			return err
		}
	}
	if payload.RecommendedPositionTargetPct != nil && payload.RecommendedPositionCapPct != nil &&
		*payload.RecommendedPositionCapPct < *payload.RecommendedPositionTargetPct {
		return fmt.Errorf("payload.recommended_position_cap_pct cannot be lower than recommended_position_target_pct")
	}
	if !payload.RecommendedTrancheStyle.IsValid() {
		return fmt.Errorf("invalid payload.recommended_tranche_style %q", payload.RecommendedTrancheStyle)
	}
	if err := validateOutputStringSlices(payload); err != nil {
		return err
	}
	if err := validateReviewChangeLog(payload.ChangeLog); err != nil {
		return err
	}
	return validateSections(payload.Sections)
}

func validateInputEnvelopeHeader(envelope InvestingReviewInputEnvelope) error {
	if envelope.SchemaVersion != AIReviewInputSchemaVersion {
		return fmt.Errorf("schema_version must be %q", AIReviewInputSchemaVersion)
	}
	if err := common.RequireString("prompt_version", envelope.PromptVersion); err != nil {
		return err
	}
	if envelope.OutputSchemaVersion != InvestingReviewOutputSchemaVersion {
		return fmt.Errorf("output_schema_version must be %q", InvestingReviewOutputSchemaVersion)
	}
	if err := common.RequireString("item_correlation_id", envelope.ItemCorrelationID); err != nil {
		return err
	}
	if err := validateRequiredObjectID("workflow_run_id", envelope.WorkflowRunID); err != nil {
		return err
	}
	if err := validateOptionalObjectID("batch_job_id", envelope.BatchJobID); err != nil {
		return err
	}
	if err := validateOptionalObjectID("batch_item_id", envelope.BatchItemID); err != nil {
		return err
	}
	if err := validateRequiredObjectID("company_id", envelope.CompanyID); err != nil {
		return err
	}
	if err := common.RequireString("symbol", envelope.Symbol); err != nil {
		return err
	}
	if envelope.BookType != common.BookTypeInvesting {
		return fmt.Errorf("book_type must be %q for investing review input", common.BookTypeInvesting)
	}
	if envelope.ReviewType != ReviewTypeInvestingCompanyReview {
		return fmt.Errorf("review_type must be %q for investing review input", ReviewTypeInvestingCompanyReview)
	}
	if err := common.RequireTime("generated_at", envelope.GeneratedAt); err != nil {
		return err
	}
	return validateRequiredObjectID("config_snapshot_id", envelope.ConfigSnapshotID)
}

func validateOutputEnvelopeHeader(envelope InvestingReviewOutputEnvelope) error {
	if envelope.SchemaVersion != AIReviewOutputEnvelopeSchemaVersion {
		return fmt.Errorf("schema_version must be %q", AIReviewOutputEnvelopeSchemaVersion)
	}
	if err := common.RequireString("prompt_version", envelope.PromptVersion); err != nil {
		return err
	}
	if envelope.OutputSchemaVersion != InvestingReviewOutputSchemaVersion {
		return fmt.Errorf("output_schema_version must be %q", InvestingReviewOutputSchemaVersion)
	}
	if err := common.RequireString("item_correlation_id", envelope.ItemCorrelationID); err != nil {
		return err
	}
	if err := validateRequiredObjectID("workflow_run_id", envelope.WorkflowRunID); err != nil {
		return err
	}
	if err := validateOptionalObjectID("batch_job_id", envelope.BatchJobID); err != nil {
		return err
	}
	if err := validateOptionalObjectID("batch_item_id", envelope.BatchItemID); err != nil {
		return err
	}
	if err := validateRequiredObjectID("company_id", envelope.CompanyID); err != nil {
		return err
	}
	if err := common.RequireString("symbol", envelope.Symbol); err != nil {
		return err
	}
	if envelope.BookType != common.BookTypeInvesting {
		return fmt.Errorf("book_type must be %q for investing review output", common.BookTypeInvesting)
	}
	if envelope.ReviewType != ReviewTypeInvestingCompanyReview {
		return fmt.Errorf("review_type must be %q for investing review output", ReviewTypeInvestingCompanyReview)
	}
	if envelope.GeneratedAt != nil && envelope.GeneratedAt.IsZero() {
		return fmt.Errorf("generated_at cannot be zero when provided")
	}
	return nil
}

func validateInvestingReviewInputPayload(payload InvestingReviewInputPayload) error {
	if err := validateRequiredObjectID("payload.company.company_id", payload.Company.CompanyID); err != nil {
		return err
	}
	if err := common.RequireString("payload.company.symbol", payload.Company.Symbol); err != nil {
		return err
	}
	if err := common.RequireString("payload.company.company_name", payload.Company.CompanyName); err != nil {
		return err
	}
	if err := validateReviewContext(payload.ReviewContext); err != nil {
		return err
	}
	if payload.CurrentPosition != nil {
		if err := validateCurrentPosition(*payload.CurrentPosition); err != nil {
			return err
		}
	}
	if len(payload.AnnualMetrics) == 0 {
		return fmt.Errorf("payload.annual_metrics must include at least one fiscal year")
	}
	for i, metric := range payload.AnnualMetrics {
		if err := common.RequireString(fmt.Sprintf("payload.annual_metrics[%d].fiscal_year", i), metric.FiscalYear); err != nil {
			return err
		}
	}
	for i, metric := range payload.QuarterlyMetrics {
		if err := common.RequireString(fmt.Sprintf("payload.quarterly_metrics[%d].period", i), metric.Period); err != nil {
			return err
		}
	}
	if err := validateValuationRanges(payload.Valuation); err != nil {
		return err
	}
	if err := validateMarketConfirmation(payload.MarketConfirmation); err != nil {
		return err
	}
	for i, evidence := range payload.TextEvidenceSummaries {
		if err := validateTextEvidenceSummary(i, evidence); err != nil {
			return err
		}
	}
	if payload.PreviousReview != nil {
		if err := validatePreviousReviewContext(*payload.PreviousReview); err != nil {
			return err
		}
	}
	return validateWeightConfig(payload.Weights)
}

func validateReviewContext(context ReviewContextInput) error {
	if err := common.RequireTime("payload.review_context.review_date", context.ReviewDate); err != nil {
		return err
	}
	if !context.Mode.IsValid() {
		return fmt.Errorf("invalid payload.review_context.mode %q", context.Mode)
	}
	if err := common.ValidatePositiveInt("payload.review_context.years_lookback", context.YearsLookback); err != nil {
		return err
	}
	if err := common.ValidatePositiveInt("payload.review_context.recent_quarter_lookback", context.RecentQuarterLookback); err != nil {
		return err
	}
	if context.CompareAgainst != compareAgainstOwnHistory {
		return fmt.Errorf("payload.review_context.compare_against must be %q", compareAgainstOwnHistory)
	}
	if context.BookType != common.BookTypeInvesting {
		return fmt.Errorf("payload.review_context.book_type must be %q", common.BookTypeInvesting)
	}
	if context.CurrentBucketBeforeReview != "" && !context.CurrentBucketBeforeReview.IsValid() {
		return fmt.Errorf("invalid payload.review_context.current_bucket_before_review %q", context.CurrentBucketBeforeReview)
	}
	if context.CurrentActionBeforeReview != "" && !context.CurrentActionBeforeReview.IsValid() {
		return fmt.Errorf("invalid payload.review_context.current_action_before_review %q", context.CurrentActionBeforeReview)
	}
	return nil
}

func validateCurrentPosition(position CurrentPositionInput) error {
	if err := common.ValidatePercentage("payload.current_position.position_pct_of_book", position.PositionPctOfBook); err != nil {
		return err
	}
	if err := common.ValidatePercentage("payload.current_position.position_pct_of_total_portfolio", position.PositionPctOfTotalPortfolio); err != nil {
		return err
	}
	if err := common.ValidatePercentage("payload.current_position.target_position_pct", position.TargetPositionPct); err != nil {
		return err
	}
	if err := common.ValidatePercentage("payload.current_position.max_position_pct", position.MaxPositionPct); err != nil {
		return err
	}
	if position.MaxPositionPct > 0 && position.TargetPositionPct > position.MaxPositionPct {
		return fmt.Errorf("payload.current_position.target_position_pct cannot exceed max_position_pct")
	}
	if position.OwnedSinceDate != nil && position.OwnedSinceDate.IsZero() {
		return fmt.Errorf("payload.current_position.owned_since_date cannot be zero")
	}
	return nil
}

func validateValuationRanges(valuation ValuationMetricsInput) error {
	ranges := map[string]ValuationRangeInput{
		"historical_pe_range":          valuation.HistoricalPERange,
		"historical_ev_ebitda_range":   valuation.HistoricalEVEBITDARange,
		"historical_pb_range":          valuation.HistoricalPBRange,
		"historical_price_sales_range": valuation.HistoricalPriceSalesRange,
		"historical_fcf_yield_range":   valuation.HistoricalFCFYieldRange,
	}
	for name, r := range ranges {
		if err := validateValuationRange("payload.valuation."+name, r); err != nil {
			return err
		}
	}
	return common.ValidateStringSlice("payload.valuation.notes", valuation.Notes)
}

func validateValuationRange(field string, r ValuationRangeInput) error {
	points := []*float64{r.Min, r.P25, r.Median, r.P75, r.Max}
	labels := []string{"min", "p25", "median", "p75", "max"}
	for i, value := range points {
		if value != nil && invalidNumber(*value) {
			return fmt.Errorf("%s.%s must be a finite number", field, labels[i])
		}
	}
	if r.CurrentPercentile != nil {
		if err := common.ValidatePercentage(field+".current_percentile", *r.CurrentPercentile); err != nil {
			return err
		}
	}
	for i := 1; i < len(points); i++ {
		if points[i-1] != nil && points[i] != nil && *points[i] < *points[i-1] {
			return fmt.Errorf("%s.%s cannot be less than %s.%s", field, labels[i], field, labels[i-1])
		}
	}
	return nil
}

func validateMarketConfirmation(market MarketConfirmationMetricsInput) error {
	if market.RelativeStrengthScore != nil {
		if err := common.ValidateComputedScore("payload.market_confirmation.relative_strength_score", *market.RelativeStrengthScore); err != nil {
			return err
		}
	}
	if market.TrendQualityScore != nil {
		if err := common.ValidateComputedScore("payload.market_confirmation.trend_quality_score", *market.TrendQualityScore); err != nil {
			return err
		}
	}
	return common.ValidateStringSlice("payload.market_confirmation.market_confirmation_notes", market.MarketConfirmationNotes)
}

func validateTextEvidenceSummary(index int, evidence TextEvidenceSummaryInput) error {
	prefix := fmt.Sprintf("payload.text_evidence_summaries[%d]", index)
	if err := common.RequireString(prefix+".source_id", evidence.SourceID); err != nil {
		return err
	}
	if !evidence.SourceType.IsValid() {
		return fmt.Errorf("invalid %s.source_type %q", prefix, evidence.SourceType)
	}
	if evidence.SourceDate != nil && evidence.SourceDate.IsZero() {
		return fmt.Errorf("%s.source_date cannot be zero", prefix)
	}
	if err := common.RequireString(prefix+".summary", evidence.Summary); err != nil {
		return err
	}
	if evidence.ConfidenceScore != nil {
		if err := common.ValidateUnitInterval(prefix+".confidence_score", *evidence.ConfidenceScore); err != nil {
			return err
		}
	}
	if err := common.ValidateStringSlice(prefix+".key_points", evidence.KeyPoints); err != nil {
		return err
	}
	if err := common.ValidateStringSlice(prefix+".risks_mentioned", evidence.RisksMentioned); err != nil {
		return err
	}
	if err := common.ValidateStringSlice(prefix+".management_claims", evidence.ManagementClaims); err != nil {
		return err
	}
	return common.ValidateStringSlice(prefix+".extracted_competitors", evidence.ExtractedCompetitors)
}

func validatePreviousReviewContext(previous PreviousReviewContextInput) error {
	if err := validateOptionalObjectID("payload.previous_review.previous_review_id", previous.PreviousReviewID); err != nil {
		return err
	}
	if previous.PreviousWeightedTotalScore != nil {
		if err := common.ValidateComputedScore("payload.previous_review.previous_weighted_total_score", *previous.PreviousWeightedTotalScore); err != nil {
			return err
		}
	}
	if previous.PreviousAction != "" && !previous.PreviousAction.IsValid() {
		return fmt.Errorf("invalid payload.previous_review.previous_action %q", previous.PreviousAction)
	}
	if previous.PreviousBucket != "" && !previous.PreviousBucket.IsValid() {
		return fmt.Errorf("invalid payload.previous_review.previous_bucket %q", previous.PreviousBucket)
	}
	if previous.PreviousThesisStatus != "" && !previous.PreviousThesisStatus.IsValid() {
		return fmt.Errorf("invalid payload.previous_review.previous_thesis_status %q", previous.PreviousThesisStatus)
	}
	for sectionName, score := range previous.PreviousSectionScores {
		if !sectionName.IsValid() {
			return fmt.Errorf("invalid payload.previous_review.previous_section_scores key %q", sectionName)
		}
		if err := common.ValidateComputedScore("payload.previous_review.previous_section_scores."+string(sectionName), score); err != nil {
			return err
		}
	}
	return nil
}

func validateWeightConfig(weights ScorecardWeightConfigInput) error {
	expectedSections := ExpectedInvestingSectionNames()
	if len(weights.SectionWeights) == 0 {
		return fmt.Errorf("payload.weights.section_weights is required")
	}
	if len(weights.SubScoreWeights) == 0 {
		return fmt.Errorf("payload.weights.sub_score_weights is required")
	}
	var sectionTotal float64
	for _, sectionName := range expectedSections {
		sectionWeight, ok := weights.SectionWeights[sectionName]
		if !ok {
			return fmt.Errorf("payload.weights.section_weights missing %q", sectionName)
		}
		if err := common.ValidatePercentage("payload.weights.section_weights."+string(sectionName), sectionWeight); err != nil {
			return err
		}
		sectionTotal += sectionWeight
		subWeights, ok := weights.SubScoreWeights[sectionName]
		if !ok {
			return fmt.Errorf("payload.weights.sub_score_weights missing section %q", sectionName)
		}
		if err := validateSubScoreWeightSet(sectionName, subWeights); err != nil {
			return err
		}
	}
	if !common.NearlyEqual(sectionTotal, 100) {
		return fmt.Errorf("payload.weights.section_weights must total 100")
	}
	return nil
}

func validateSubScoreWeightSet(sectionName common.SectionName, weights map[common.SubScoreName]float64) error {
	expected := ExpectedSubScoreNamesBySection()[sectionName]
	if len(weights) == 0 {
		return fmt.Errorf("payload.weights.sub_score_weights.%s is required", sectionName)
	}
	var total float64
	for _, subScoreName := range expected {
		weight, ok := weights[subScoreName]
		if !ok {
			return fmt.Errorf("payload.weights.sub_score_weights.%s missing %q", sectionName, subScoreName)
		}
		if err := common.ValidatePercentage("payload.weights.sub_score_weights."+string(sectionName)+"."+string(subScoreName), weight); err != nil {
			return err
		}
		total += weight
	}
	if !common.NearlyEqual(total, 100) {
		return fmt.Errorf("payload.weights.sub_score_weights.%s must total 100", sectionName)
	}
	return nil
}

func validateSections(sections []AISectionScoreOutput) error {
	expectedSections := ExpectedInvestingSectionNames()
	if len(sections) != len(expectedSections) {
		return fmt.Errorf("payload.sections must contain exactly %d sections", len(expectedSections))
	}
	seen := make(map[common.SectionName]struct{}, len(sections))
	var totalWeight float64
	for i, section := range sections {
		if _, exists := seen[section.SectionName]; exists {
			return fmt.Errorf("duplicate payload.sections[%d].section_name %q", i, section.SectionName)
		}
		seen[section.SectionName] = struct{}{}
		if err := validateSection(i, section); err != nil {
			return err
		}
		totalWeight += section.SectionWeight
	}
	for _, sectionName := range expectedSections {
		if _, ok := seen[sectionName]; !ok {
			return fmt.Errorf("payload.sections missing section %q", sectionName)
		}
	}
	if !common.NearlyEqual(totalWeight, 100) {
		return fmt.Errorf("payload.sections section_weight values must total 100")
	}
	return nil
}

func validateSection(index int, section AISectionScoreOutput) error {
	prefix := fmt.Sprintf("payload.sections[%d]", index)
	if !section.SectionName.IsValid() {
		return fmt.Errorf("invalid %s.section_name %q", prefix, section.SectionName)
	}
	if err := common.ValidatePercentage(prefix+".section_weight", section.SectionWeight); err != nil {
		return err
	}
	if err := common.ValidateScore(prefix+".section_score_raw", section.SectionScoreRaw); err != nil {
		return err
	}
	if section.SectionScoreWeighted != nil {
		if err := common.ValidateComputedScore(prefix+".section_score_weighted", *section.SectionScoreWeighted); err != nil {
			return err
		}
		expected := common.NormalizeWeightedScore(section.SectionScoreRaw, section.SectionWeight)
		if !common.NearlyEqual(expected, *section.SectionScoreWeighted) {
			return fmt.Errorf("%s.section_score_weighted does not match raw score and weight", prefix)
		}
	}
	if !section.SectionActionCap.IsValid() {
		return fmt.Errorf("invalid %s.section_action_cap %q", prefix, section.SectionActionCap)
	}
	if err := common.RequireString(prefix+".section_summary", section.SectionSummary); err != nil {
		return err
	}
	if err := common.ValidateStringSlice(prefix+".section_strengths", section.SectionStrengths); err != nil {
		return err
	}
	if err := common.ValidateStringSlice(prefix+".section_weaknesses", section.SectionWeaknesses); err != nil {
		return err
	}
	if err := common.ValidateStringSlice(prefix+".section_risks", section.SectionRisks); err != nil {
		return err
	}
	if err := common.ValidateUnitInterval(prefix+".section_confidence_score", section.SectionConfidenceScore); err != nil {
		return err
	}
	if err := validateSubScores(prefix, section.SectionName, section.SubScores); err != nil {
		return err
	}
	if err := validateEvidenceRefs(prefix, section.EvidenceRefs); err != nil {
		return err
	}
	return validateSubScoreEvidenceReferences(prefix, section.SubScores, section.EvidenceRefs)
}

func validateSubScores(prefix string, sectionName common.SectionName, subScores []AISubScoreOutput) error {
	expected := ExpectedSubScoreNamesBySection()[sectionName]
	if len(subScores) != len(expected) {
		return fmt.Errorf("%s.sub_scores must contain exactly %d sub-scores", prefix, len(expected))
	}
	allowed := make(map[common.SubScoreName]struct{}, len(expected))
	for _, name := range expected {
		allowed[name] = struct{}{}
	}
	seen := make(map[common.SubScoreName]struct{}, len(subScores))
	var weightTotal float64
	for i, subScore := range subScores {
		subPrefix := fmt.Sprintf("%s.sub_scores[%d]", prefix, i)
		if !subScore.SubScoreName.IsValid() {
			return fmt.Errorf("invalid %s.sub_score_name %q", subPrefix, subScore.SubScoreName)
		}
		if _, ok := allowed[subScore.SubScoreName]; !ok {
			return fmt.Errorf("%s.sub_score_name %q does not belong to section %q", subPrefix, subScore.SubScoreName, sectionName)
		}
		if _, exists := seen[subScore.SubScoreName]; exists {
			return fmt.Errorf("duplicate %s.sub_score_name %q", subPrefix, subScore.SubScoreName)
		}
		seen[subScore.SubScoreName] = struct{}{}
		if err := common.ValidatePercentage(subPrefix+".sub_score_weight", subScore.SubScoreWeight); err != nil {
			return err
		}
		weightTotal += subScore.SubScoreWeight
		if err := common.ValidateScore(subPrefix+".sub_score_value", subScore.SubScoreValue); err != nil {
			return err
		}
		if err := common.RequireString(subPrefix+".sub_score_summary", subScore.SubScoreSummary); err != nil {
			return err
		}
		if !subScore.TrendDirection.IsValid() {
			return fmt.Errorf("invalid %s.trend_direction %q", subPrefix, subScore.TrendDirection)
		}
		if !subScore.EvidenceStrength.IsValid() || subScore.EvidenceStrength == "" {
			return fmt.Errorf("invalid %s.evidence_strength %q", subPrefix, subScore.EvidenceStrength)
		}
		if !subScore.MetricBasis.IsValid() {
			return fmt.Errorf("invalid %s.metric_basis %q", subPrefix, subScore.MetricBasis)
		}
		if err := validateStringIDs(subPrefix+".evidence_ref_ids", subScore.EvidenceRefIDs); err != nil {
			return err
		}
	}
	if !common.NearlyEqual(weightTotal, 100) {
		return fmt.Errorf("%s.sub_scores sub_score_weight values must total 100", prefix)
	}
	return nil
}

func validateEvidenceRefs(prefix string, evidenceRefs []AIEvidenceReferenceOutput) error {
	seen := make(map[string]struct{}, len(evidenceRefs))
	for i, evidence := range evidenceRefs {
		evidencePrefix := fmt.Sprintf("%s.evidence_refs[%d]", prefix, i)
		if err := common.RequireString(evidencePrefix+".evidence_id", evidence.EvidenceID); err != nil {
			return err
		}
		if _, exists := seen[evidence.EvidenceID]; exists {
			return fmt.Errorf("duplicate %s.evidence_id %q", evidencePrefix, evidence.EvidenceID)
		}
		seen[evidence.EvidenceID] = struct{}{}
		if !evidence.SourceType.IsValid() {
			return fmt.Errorf("invalid %s.source_type %q", evidencePrefix, evidence.SourceType)
		}
		if evidence.SourceDate != nil && evidence.SourceDate.IsZero() {
			return fmt.Errorf("%s.source_date cannot be zero", evidencePrefix)
		}
		if err := common.RequireString(evidencePrefix+".evidence_summary", evidence.EvidenceSummary); err != nil {
			return err
		}
		if !evidence.EvidenceDirection.IsValid() || evidence.EvidenceDirection == "" {
			return fmt.Errorf("invalid %s.evidence_direction %q", evidencePrefix, evidence.EvidenceDirection)
		}
		if evidence.SourceTitle == "" && evidence.SourceURLOrPath == "" && evidence.ExcerptOrMetricName == "" {
			return fmt.Errorf("%s must include source_title, source_url_or_path, or excerpt_or_metric_name", evidencePrefix)
		}
	}
	return nil
}

func validateSubScoreEvidenceReferences(prefix string, subScores []AISubScoreOutput, evidenceRefs []AIEvidenceReferenceOutput) error {
	available := make(map[string]struct{}, len(evidenceRefs))
	for _, evidence := range evidenceRefs {
		available[evidence.EvidenceID] = struct{}{}
	}
	for i, subScore := range subScores {
		for j, evidenceRefID := range subScore.EvidenceRefIDs {
			if _, ok := available[evidenceRefID]; !ok {
				return fmt.Errorf("%s.sub_scores[%d].evidence_ref_ids[%d] references unknown evidence id %q", prefix, i, j, evidenceRefID)
			}
		}
	}
	return nil
}

func validateReviewChangeLog(changeLog AIReviewChangeLogOutput) error {
	if err := validateOptionalObjectID("payload.change_log.previous_review_id", changeLog.PreviousReviewID); err != nil {
		return err
	}
	if changeLog.WeightedTotalScoreChange != nil && invalidNumber(*changeLog.WeightedTotalScoreChange) {
		return fmt.Errorf("payload.change_log.weighted_total_score_change must be finite")
	}
	for sectionName := range changeLog.SectionScoreChanges {
		if !sectionName.IsValid() {
			return fmt.Errorf("invalid payload.change_log.section_score_changes key %q", sectionName)
		}
	}
	for subScoreName := range changeLog.SubScoreChanges {
		if !subScoreName.IsValid() {
			return fmt.Errorf("invalid payload.change_log.sub_score_changes key %q", subScoreName)
		}
	}
	if err := common.ValidateStringSlice("payload.change_log.major_positive_changes", changeLog.MajorPositiveChanges); err != nil {
		return err
	}
	if err := common.ValidateStringSlice("payload.change_log.major_negative_changes", changeLog.MajorNegativeChanges); err != nil {
		return err
	}
	return nil
}

func validateOutputStringSlices(payload InvestingReviewOutputPayload) error {
	checks := map[string][]string{
		"payload.action_constraints":            payload.ActionConstraints,
		"payload.key_growth_drivers":            payload.KeyGrowthDrivers,
		"payload.key_moat_or_advantage_factors": payload.KeyMoatOrAdvantageFactors,
		"payload.key_risks":                     payload.KeyRisks,
		"payload.disconfirming_signals":         payload.DisconfirmingSignals,
		"payload.what_would_break_the_thesis":   payload.WhatWouldBreakTheThesis,
		"payload.missing_data_points":           payload.MissingDataPoints,
		"payload.low_confidence_areas":          payload.LowConfidenceAreas,
		"payload.assumptions_made":              payload.AssumptionsMade,
		"payload.warnings":                      payload.Warnings,
	}
	for field, values := range checks {
		if err := common.ValidateStringSlice(field, values); err != nil {
			return err
		}
	}
	return nil
}

func validateRequiredObjectID(field string, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	return validateObjectID(field, value)
}

func validateOptionalObjectID(field string, value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return validateObjectID(field, value)
}

func validateObjectID(field string, value string) error {
	if _, err := primitive.ObjectIDFromHex(strings.TrimSpace(value)); err != nil {
		return fmt.Errorf("%s must be a valid object id: %w", field, err)
	}
	return nil
}

func validateStringIDs(field string, values []string) error {
	for i, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s[%d] cannot be blank", field, i)
		}
	}
	return nil
}

func invalidNumber(value float64) bool {
	return math.IsNaN(value) || math.IsInf(value, 0)
}
