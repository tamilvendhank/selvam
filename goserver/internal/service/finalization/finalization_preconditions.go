package finalization

import (
	"fmt"

	domaincommon "goserver/internal/domain/common"
	domainreview "goserver/internal/domain/review"
)

func validateFinalizationEligibility(review *domainreview.CompanyReview, options finalizationRequestOptions) error {
	if review == nil {
		return fmt.Errorf("review is required")
	}
	if !options.WorkflowRunID.IsZero() && review.WorkflowRunID != options.WorkflowRunID {
		return fmt.Errorf("%w: workflowRunId filter does not match", errFinalizationSkipped)
	}
	if !options.CompanyID.IsZero() && review.CompanyID != options.CompanyID {
		return fmt.Errorf("%w: companyId filter does not match", errFinalizationSkipped)
	}
	if options.BookType != "" && review.BookType != options.BookType {
		return fmt.Errorf("%w: bookType filter does not match", errFinalizationSkipped)
	}
	if isSuperseded(review) {
		return fmt.Errorf("%w: review is already superseded", errFinalizationSkipped)
	}
	if isAlreadyFinalized(review) {
		return fmt.Errorf("%w: review is already finalized", errFinalizationSkipped)
	}
	if review.ReviewLifecycleState != domaincommon.ReviewLifecycleStateAIValidated {
		return fmt.Errorf("%w: lifecycle state %q is not finalizable", errFinalizationSkipped, review.ReviewLifecycleState)
	}
	if review.ReviewStatus != domaincommon.ReviewStatusDraft {
		return fmt.Errorf("%w: review status %q is not finalizable", errFinalizationSkipped, review.ReviewStatus)
	}
	if err := hasRequiredFinalizableContent(review); err != nil {
		return err
	}
	if !review.CanFinalize() {
		return fmt.Errorf("review cannot be finalized from lifecycle %q status %q", review.ReviewLifecycleState, review.ReviewStatus)
	}
	return nil
}

func hasRequiredFinalizableContent(review *domainreview.CompanyReview) error {
	if review.CompanyID.IsZero() {
		return fmt.Errorf("companyId is required for review finalization")
	}
	if !review.BookType.IsValid() {
		return fmt.Errorf("valid bookType is required for review finalization")
	}
	if review.ConfigSnapshotID.IsZero() {
		return fmt.Errorf("configSnapshotId is required for review finalization")
	}
	if len(review.Sections) == 0 {
		return fmt.Errorf("sections are required for review finalization")
	}
	if review.DecisionAction == nil {
		return fmt.Errorf("decisionAction is required for review finalization")
	}
	if review.FinalActionAfterReview == "" {
		return fmt.Errorf("finalActionAfterReview is required for review finalization")
	}
	if review.BookType == domaincommon.BookTypeInvesting && review.FinalBucketAfterReview == "" {
		return fmt.Errorf("finalBucketAfterReview is required for investing review finalization")
	}
	// This delegates the detailed score, section, action, bucket, and timestamp rules
	// to the domain model before the repository performs the guarded transition.
	if err := review.Validate(); err != nil {
		return fmt.Errorf("review content is not finalizable: %w", err)
	}
	return nil
}

func isAlreadyFinalized(review *domainreview.CompanyReview) bool {
	return review != nil &&
		review.ReviewLifecycleState == domaincommon.ReviewLifecycleStateFinalized &&
		review.ReviewStatus == domaincommon.ReviewStatusFinal
}

func isSuperseded(review *domainreview.CompanyReview) bool {
	return review != nil &&
		(review.ReviewLifecycleState == domaincommon.ReviewLifecycleStateSuperseded ||
			review.ReviewStatus == domaincommon.ReviewStatusSuperseded)
}
