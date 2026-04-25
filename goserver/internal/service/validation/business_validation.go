package validation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domainaijob "goserver/internal/domain/aijob"
	domaincommon "goserver/internal/domain/common"
	platformrepo "goserver/internal/platform/repository"
)

func (service *aiOutputValidationService) validateCompanyReviewPayload(
	ctx context.Context,
	item *domainaijob.AIBatchItem,
	payload map[string]any,
	options validationRequestOptions,
) outputValidationReport {
	report := outputValidationReport{}
	if item.TargetReviewID.IsZero() {
		report.Add(issueError("missing_review_link", "targetReviewId", "company review batch item must reference a review shell", item))
	} else if service.reviews != nil {
		if _, err := service.reviews.GetByID(ctx, item.TargetReviewID); err != nil {
			if errors.Is(err, platformrepo.ErrNotFound) {
				report.Add(issueError("review_not_found", "targetReviewId", "linked review shell was not found", item))
			} else {
				report.Add(issueWarning("review_lookup_failed", "targetReviewId", fmt.Sprintf("linked review shell lookup failed: %v", err), item))
			}
		}
	}

	weightedTotal := requiredFloatInRange(&report, payload, "weightedTotalScore", 0, 10, item, "weightedTotalScore", "weighted_total_score")
	optionalFloatInRange(&report, payload, "confidenceScore", 0, 1, item, "confidenceScore", "confidence_score")
	optionalStringEnum(&report, payload, "finalActionAfterReview", item, isValidAction, "finalActionAfterReview", "final_action_after_review", "actionType", "action")
	optionalStringEnum(&report, payload, "finalBucketAfterReview", item, isValidBucket, "finalBucketAfterReview", "final_bucket_after_review", "bucketAfterAction", "bucket")

	sections, present, ok := getArray(payload, "sections", "sectionScores", "section_scores")
	if !present {
		report.Add(issueMissingField("sections", item))
		return report
	}
	if !ok {
		report.Add(issueTypeMismatch("sections", "an array", item))
		return report
	}
	if len(sections) == 0 {
		report.Add(issueError("empty_sections", "sections", "sections must include at least one section", item))
		return report
	}

	sectionScores := service.validateCompanyReviewSections(&report, item, sections, options)
	if len(sectionScores) > 0 {
		checkWeightedScoreSanity(&report, item, weightedTotal, sectionScores, options.StrictMode)
	}
	return report
}

func (service *aiOutputValidationService) validateCompanyReviewSections(
	report *outputValidationReport,
	item *domainaijob.AIBatchItem,
	sections []any,
	options validationRequestOptions,
) map[domaincommon.SectionName]sectionScoreForSanity {
	seen := make(map[domaincommon.SectionName]struct{}, len(sections))
	sectionScores := make(map[domaincommon.SectionName]sectionScoreForSanity, len(sections))
	for index, rawSection := range sections {
		path := fmt.Sprintf("sections[%d]", index)
		section, ok := rawSection.(map[string]any)
		if !ok {
			report.Add(issueTypeMismatch(path, "an object", item))
			continue
		}

		nameValue := requiredString(report, section, path+".sectionName", item, "sectionName", "section_name", "name")
		sectionName := domaincommon.SectionName(strings.TrimSpace(nameValue))
		if nameValue != "" && !sectionName.IsValid() {
			report.Add(issueUnknownSection(path+".sectionName", nameValue, item, options.StrictMode))
		}
		if sectionName.IsValid() {
			if _, exists := seen[sectionName]; exists {
				report.Add(issueDuplicate(path+".sectionName", string(sectionName), item))
			}
			seen[sectionName] = struct{}{}
		}

		rawScore := requiredFloatInRange(report, section, path+".sectionScoreRaw", 1, 10, item, "sectionScoreRaw", "section_score_raw", "sectionScore", "section_score", "score")
		weight, hasWeight := optionalFloatInRange(report, section, path+".sectionWeight", 0, 100, item, "sectionWeight", "section_weight", "weight")
		if _, ok := optionalFloatInRange(report, section, path+".sectionScoreWeighted", 0, 10, item, "sectionScoreWeighted", "section_score_weighted", "weightedScore", "weighted_score"); ok && hasWeight {
			// A full recomputation belongs in materialization; here we only make the supplied fields parseable.
		}
		optionalFloatInRange(report, section, path+".sectionConfidenceScore", 0, 1, item, "sectionConfidenceScore", "section_confidence_score", "confidenceScore", "confidence_score")
		optionalStringEnum(report, section, path+".sectionActionCap", item, isValidSectionActionCap, "sectionActionCap", "section_action_cap", "actionCap", "action_cap")

		service.validateSectionSubScores(report, item, sectionName, section, path, options)
		validateEvidenceRefs(report, item, section, path+".evidenceRefs", "evidenceRefs", "evidence_refs")

		if sectionName.IsValid() {
			sectionScores[sectionName] = sectionScoreForSanity{
				RawScore:  rawScore,
				Weight:    weight,
				HasWeight: hasWeight,
			}
		}
	}

	for expected := range domaincommon.DefaultSectionWeights {
		if _, exists := seen[expected]; !exists {
			addStrictnessIssue(report, options.StrictMode, "missing_section", "sections", fmt.Sprintf("missing expected section %q", expected), item)
		}
	}
	return sectionScores
}

func (service *aiOutputValidationService) validateSectionSubScores(
	report *outputValidationReport,
	item *domainaijob.AIBatchItem,
	sectionName domaincommon.SectionName,
	section map[string]any,
	path string,
	options validationRequestOptions,
) {
	subScores, present, ok := getArray(section, "subScores", "sub_scores")
	if !present {
		report.Add(issueMissingField(path+".subScores", item))
		return
	}
	if !ok {
		report.Add(issueTypeMismatch(path+".subScores", "an array", item))
		return
	}
	if len(subScores) == 0 {
		report.Add(issueError("empty_sub_scores", path+".subScores", "section must include sub-scores", item))
		return
	}

	expected := domaincommon.DefaultSubScoreWeights[sectionName]
	seen := make(map[domaincommon.SubScoreName]struct{}, len(subScores))
	var weightTotal float64
	hasWeights := false
	for index, rawSubScore := range subScores {
		subPath := fmt.Sprintf("%s.subScores[%d]", path, index)
		subScore, ok := rawSubScore.(map[string]any)
		if !ok {
			report.Add(issueTypeMismatch(subPath, "an object", item))
			continue
		}

		nameValue := requiredString(report, subScore, subPath+".subScoreName", item, "subScoreName", "sub_score_name", "name")
		subScoreName := domaincommon.SubScoreName(strings.TrimSpace(nameValue))
		if nameValue != "" && !subScoreName.IsValid() {
			report.Add(issueUnknownSubScore(subPath+".subScoreName", nameValue, item, options.StrictMode))
		}
		if subScoreName.IsValid() {
			if _, exists := seen[subScoreName]; exists {
				report.Add(issueDuplicate(subPath+".subScoreName", string(subScoreName), item))
			}
			seen[subScoreName] = struct{}{}
			if len(expected) > 0 {
				if _, belongs := expected[subScoreName]; !belongs {
					addStrictnessIssue(report, options.StrictMode, "sub_score_section_mismatch", subPath+".subScoreName", fmt.Sprintf("sub-score %q does not belong to section %q", subScoreName, sectionName), item)
				}
			}
		}

		requiredFloatInRange(report, subScore, subPath+".subScoreValue", 1, 10, item, "subScoreValue", "sub_score_value", "value", "score")
		if weight, present := optionalFloatInRange(report, subScore, subPath+".subScoreWeight", 0, 100, item, "subScoreWeight", "sub_score_weight", "weight"); present {
			hasWeights = true
			weightTotal += weight
		}
		optionalStringEnum(report, subScore, subPath+".trendDirection", item, isValidTrendDirection, "trendDirection", "trend_direction")
		optionalStringEnum(report, subScore, subPath+".metricBasis", item, isValidMetricBasis, "metricBasis", "metric_basis")
		optionalStringEnum(report, subScore, subPath+".evidenceStrength", item, isValidEvidenceStrength, "evidenceStrength", "evidence_strength")
		validateEvidenceRefs(report, item, subScore, subPath+".evidenceRefs", "evidenceRefs", "evidence_refs", "evidenceRefIDs", "evidence_ref_ids")
	}

	if hasWeights && (weightTotal < 99.5 || weightTotal > 100.5) {
		addStrictnessIssue(report, options.StrictMode, "sub_score_weight_total", path+".subScores", fmt.Sprintf("sub-score weights should total 100, got %.2f", weightTotal), item)
	}
	for expectedName := range expected {
		if _, exists := seen[expectedName]; !exists {
			addStrictnessIssue(report, options.StrictMode, "missing_sub_score", path+".subScores", fmt.Sprintf("missing expected sub-score %q", expectedName), item)
		}
	}
}

func validateEvidenceRefs(
	report *outputValidationReport,
	item *domainaijob.AIBatchItem,
	payload map[string]any,
	path string,
	aliases ...string,
) {
	refs, present, ok := getArray(payload, aliases...)
	if !present {
		return
	}
	if !ok {
		report.Add(issueTypeMismatch(path, "an array", item))
		return
	}
	for index, rawRef := range refs {
		refPath := fmt.Sprintf("%s[%d]", path, index)
		ref, ok := rawRef.(map[string]any)
		if !ok {
			if _, stringOK := rawRef.(string); stringOK {
				continue
			}
			report.Add(issueTypeMismatch(refPath, "an object or string reference", item))
			continue
		}
		optionalStringEnum(report, ref, refPath+".sourceType", item, isValidEvidenceSourceType, "sourceType", "source_type")
		optionalStringEnum(report, ref, refPath+".evidenceDirection", item, isValidEvidenceDirection, "evidenceDirection", "evidence_direction")
		if _, present, ok := getString(ref, "sourceTitle", "source_title", "sourceURLOrPath", "source_url_or_path", "excerptOrMetricName", "excerpt_or_metric_name"); present && !ok {
			report.Add(issueTypeMismatch(refPath, "evidence reference text fields must be strings", item))
		}
	}
}

type sectionScoreForSanity struct {
	RawScore  float64
	Weight    float64
	HasWeight bool
}

func checkWeightedScoreSanity(
	report *outputValidationReport,
	item *domainaijob.AIBatchItem,
	weightedTotal float64,
	sections map[domaincommon.SectionName]sectionScoreForSanity,
	strict bool,
) {
	var computed float64
	var weightTotal float64
	hasAnyWeight := false
	for sectionName, section := range sections {
		weight := section.Weight
		if !section.HasWeight {
			weight = domaincommon.DefaultSectionWeights[sectionName]
		} else {
			hasAnyWeight = true
		}
		weightTotal += weight
		computed += domaincommon.NormalizeWeightedScore(section.RawScore, weight)
	}
	if hasAnyWeight && (weightTotal < 99.5 || weightTotal > 100.5) {
		addStrictnessIssue(report, strict, "section_weight_total", "sections", fmt.Sprintf("section weights should total 100, got %.2f", weightTotal), item)
	}
	if computed > 0 && (weightedTotal-computed > 1.5 || computed-weightedTotal > 1.5) {
		addStrictnessIssue(report, strict, "weighted_score_inconsistent", "weightedTotalScore", fmt.Sprintf("weightedTotalScore %.2f differs materially from section score sanity estimate %.2f", weightedTotal, computed), item)
	}
}

func validateThesisUpdatePayload(
	item *domainaijob.AIBatchItem,
	payload map[string]any,
	options validationRequestOptions,
) outputValidationReport {
	report := validateGenericStructuredPayload(item, payload, "thesis update", []string{"thesis", "thesisUpdate", "thesis_update", "summary", "changeSummary", "change_summary"})
	optionalStringEnum(&report, payload, "thesisStatus", item, func(value string) bool {
		return domaincommon.ThesisStatus(strings.TrimSpace(value)).IsValid()
	}, "thesisStatus", "thesis_status")
	return report
}

func validateChangeSummaryPayload(
	item *domainaijob.AIBatchItem,
	payload map[string]any,
	options validationRequestOptions,
) outputValidationReport {
	return validateGenericStructuredPayload(item, payload, "change summary", []string{"changes", "summary", "changeSummary", "change_summary", "whatChangedSummary", "what_changed_summary"})
}

func validateEvidenceSummaryPayload(
	item *domainaijob.AIBatchItem,
	payload map[string]any,
	options validationRequestOptions,
) outputValidationReport {
	return validateGenericStructuredPayload(item, payload, "evidence summary", []string{"evidence", "evidenceSummary", "evidence_summary", "summary", "sources"})
}

func validateTradingCandidateReviewPayload(
	item *domainaijob.AIBatchItem,
	payload map[string]any,
	options validationRequestOptions,
) outputValidationReport {
	report := validateGenericStructuredPayload(item, payload, "trading candidate review", []string{"decision", "action", "summary", "rationale", "tradingCandidateReview", "trading_candidate_review"})
	optionalStringEnum(&report, payload, "action", item, isValidAction, "action", "actionType", "action_type")
	return report
}

func validateGenericStructuredPayload(
	item *domainaijob.AIBatchItem,
	payload map[string]any,
	label string,
	meaningfulFields []string,
) outputValidationReport {
	report := outputValidationReport{}
	if len(payload) == 0 {
		report.Add(issueError("empty_payload", "resultPayload", fmt.Sprintf("%s payload is empty", label), item))
		return report
	}
	foundMeaningfulField := false
	for _, field := range meaningfulFields {
		if value, ok := payload[field]; ok && value != nil && strings.TrimSpace(fmt.Sprint(value)) != "" {
			foundMeaningfulField = true
			break
		}
	}
	if !foundMeaningfulField {
		report.Add(issueError("missing_required_shape", "resultPayload", fmt.Sprintf("%s payload must include one of: %s", label, strings.Join(meaningfulFields, ", ")), item))
	}
	if version, present := optionalFloatInRange(&report, payload, "schemaVersion", 1, 1000, item, "schemaVersion", "schema_version"); present && version != float64(int(version)) {
		report.Add(issueError("invalid_schema_version", "schemaVersion", "schemaVersion must be an integer", item))
	}
	return report
}
