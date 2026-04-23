package common

// SchemaVersion1 is the persisted schema version used by the V1 domain model.
const SchemaVersion1 = 1

type BookType string

const (
	BookTypeInvesting BookType = "investing"
	BookTypeTrading   BookType = "trading"
)

type InvestingMode string

const (
	InvestingModeEarlyHunter         InvestingMode = "early_hunter"
	InvestingModeBalanced            InvestingMode = "balanced"
	InvestingModeConfirmedCompounder InvestingMode = "confirmed_compounder"
)

type ReviewPeriodType string

const (
	ReviewPeriodTypeMonthly     ReviewPeriodType = "monthly"
	ReviewPeriodTypeQuarterly   ReviewPeriodType = "quarterly"
	ReviewPeriodTypeEventDriven ReviewPeriodType = "event_driven"
	ReviewPeriodTypeManual      ReviewPeriodType = "manual"
)

// ReviewStatus is the business-facing review state.
// It remains coarse by design and should stay consistent with ReviewLifecycleState.
type ReviewStatus string

const (
	ReviewStatusDraft      ReviewStatus = "draft"
	ReviewStatusFinal      ReviewStatus = "final"
	ReviewStatusSuperseded ReviewStatus = "superseded"
)

// ReviewLifecycleState captures the async production lifecycle of a review shell.
type ReviewLifecycleState string

const (
	ReviewLifecycleStatePendingInput           ReviewLifecycleState = "pending_input"
	ReviewLifecycleStatePendingAI              ReviewLifecycleState = "pending_ai"
	ReviewLifecycleStateAICompletedUnvalidated ReviewLifecycleState = "ai_completed_unvalidated"
	ReviewLifecycleStateValidationFailed       ReviewLifecycleState = "validation_failed"
	ReviewLifecycleStateAIValidated            ReviewLifecycleState = "ai_validated"
	ReviewLifecycleStateFinalized              ReviewLifecycleState = "finalized"
	ReviewLifecycleStateSuperseded             ReviewLifecycleState = "superseded"
)

type ReviewerType string

const (
	ReviewerTypeAI     ReviewerType = "ai"
	ReviewerTypeHuman  ReviewerType = "human"
	ReviewerTypeHybrid ReviewerType = "hybrid"
)

type WatchlistBucket string

const (
	WatchlistBucketResearch   WatchlistBucket = "research"
	WatchlistBucketWatch      WatchlistBucket = "watch"
	WatchlistBucketBuyReady   WatchlistBucket = "buy_ready"
	WatchlistBucketHold       WatchlistBucket = "hold"
	WatchlistBucketExitReview WatchlistBucket = "exit_review"
)

type InvestingActionType string

const (
	InvestingActionTypeBuy    InvestingActionType = "buy"
	InvestingActionTypeWatch  InvestingActionType = "watch"
	InvestingActionTypeHold   InvestingActionType = "hold"
	InvestingActionTypeTrim   InvestingActionType = "trim"
	InvestingActionTypeSell   InvestingActionType = "sell"
	InvestingActionTypeReject InvestingActionType = "reject"
)

type SectionName string

const (
	SectionNameInvestability                      SectionName = "investability"
	SectionNameBusinessTraction                   SectionName = "business_traction"
	SectionNameProfitConversion                   SectionName = "profit_conversion"
	SectionNameCapitalEfficiencyFinancialStrength SectionName = "capital_efficiency_financial_strength"
	SectionNameStructuralSectorAttractiveness     SectionName = "structural_sector_attractiveness"
	SectionNameRunwayIndustryPositioning          SectionName = "runway_industry_positioning"
	SectionNameManagementGovernance               SectionName = "management_governance"
	SectionNameMarketConfirmation                 SectionName = "market_confirmation"
	SectionNameValuationEntryAttractiveness       SectionName = "valuation_entry_attractiveness"
)

type SectionActionCap string

const (
	SectionActionCapCannotBuy      SectionActionCap = "cannot_buy"
	SectionActionCapWatchOnly      SectionActionCap = "watch_only"
	SectionActionCapExitReviewOnly SectionActionCap = "exit_review_only"
	SectionActionCapNone           SectionActionCap = "none"
)

type TrendDirection string

const (
	TrendDirectionImproving TrendDirection = "improving"
	TrendDirectionStable    TrendDirection = "stable"
	TrendDirectionWeakening TrendDirection = "weakening"
	TrendDirectionMixed     TrendDirection = "mixed"
)

type MetricBasis string

const (
	MetricBasisQuant  MetricBasis = "quant"
	MetricBasisText   MetricBasis = "text"
	MetricBasisHybrid MetricBasis = "hybrid"
)

type EvidenceStrength string

const (
	EvidenceStrengthLow    EvidenceStrength = "low"
	EvidenceStrengthMedium EvidenceStrength = "medium"
	EvidenceStrengthHigh   EvidenceStrength = "high"
)

type EvidenceSourceType string

const (
	EvidenceSourceTypeAnnualReport         EvidenceSourceType = "annual_report"
	EvidenceSourceTypeConcall              EvidenceSourceType = "concall"
	EvidenceSourceTypeInvestorPresentation EvidenceSourceType = "investor_presentation"
	EvidenceSourceTypeExchangeFiling       EvidenceSourceType = "exchange_filing"
	EvidenceSourceTypeFinancialData        EvidenceSourceType = "financial_data"
	EvidenceSourceTypePriceData            EvidenceSourceType = "price_data"
	EvidenceSourceTypeManualNote           EvidenceSourceType = "manual_note"
)

type EvidenceDirection string

const (
	EvidenceDirectionPositive EvidenceDirection = "positive"
	EvidenceDirectionNegative EvidenceDirection = "negative"
	EvidenceDirectionNeutral  EvidenceDirection = "neutral"
)

type ThesisStatus string

const (
	ThesisStatusActive      ThesisStatus = "active"
	ThesisStatusUnderReview ThesisStatus = "under_review"
	ThesisStatusBroken      ThesisStatus = "broken"
	ThesisStatusArchived    ThesisStatus = "archived"
)

type PositionRole string

const (
	PositionRoleStarter       PositionRole = "starter"
	PositionRoleBuilding      PositionRole = "building"
	PositionRoleCore          PositionRole = "core"
	PositionRoleTrimCandidate PositionRole = "trim_candidate"
	PositionRoleExitCandidate PositionRole = "exit_candidate"
)

type WorkflowRunType string

const (
	WorkflowRunTypeMonthlyScan      WorkflowRunType = "monthly_scan"
	WorkflowRunTypeQuarterlyRefresh WorkflowRunType = "quarterly_refresh"
	WorkflowRunTypeEventRefresh     WorkflowRunType = "event_refresh"
	WorkflowRunTypeManual           WorkflowRunType = "manual"
)

type WorkflowRunStatus string

const (
	WorkflowRunStatusCreated            WorkflowRunStatus = "created"
	WorkflowRunStatusRunning            WorkflowRunStatus = "running"
	WorkflowRunStatusWaitingExternal    WorkflowRunStatus = "waiting_external"
	WorkflowRunStatusPartiallyCompleted WorkflowRunStatus = "partially_completed"
	WorkflowRunStatusCompleted          WorkflowRunStatus = "completed"
	WorkflowRunStatusFailed             WorkflowRunStatus = "failed"
	WorkflowRunStatusCancelled          WorkflowRunStatus = "cancelled"
)

type WorkflowStepName string

const (
	WorkflowStepNameScanUniverse                   WorkflowStepName = "scan_universe"
	WorkflowStepNameApplyHardFilters               WorkflowStepName = "apply_hard_filters"
	WorkflowStepNameBuildReviewInputs              WorkflowStepName = "build_review_inputs"
	WorkflowStepNameCreatePendingReviewRecords     WorkflowStepName = "create_pending_review_records"
	WorkflowStepNameCreateBatchJob                 WorkflowStepName = "create_batch_job"
	WorkflowStepNameSubmitBatchJob                 WorkflowStepName = "submit_batch_job"
	WorkflowStepNameWaitForAsyncResults            WorkflowStepName = "wait_for_async_results"
	WorkflowStepNamePollAndReconcileBatchResults   WorkflowStepName = "poll_and_reconcile_batch_results"
	WorkflowStepNameValidateAIOutputs              WorkflowStepName = "validate_ai_outputs"
	WorkflowStepNameMaterializeFinalReviews        WorkflowStepName = "materialize_final_reviews"
	WorkflowStepNameEvaluateThesisAndChange        WorkflowStepName = "evaluate_thesis_and_change"
	WorkflowStepNameMapActions                     WorkflowStepName = "map_actions"
	WorkflowStepNameAssignBuckets                  WorkflowStepName = "assign_buckets"
	WorkflowStepNameBuildCapitalCandidates         WorkflowStepName = "build_capital_candidates"
	WorkflowStepNameAllocateCapital                WorkflowStepName = "allocate_capital"
	WorkflowStepNamePersistOutputs                 WorkflowStepName = "persist_outputs"
	WorkflowStepNamePublishRunSummary              WorkflowStepName = "publish_run_summary"
	WorkflowStepNameRefreshUniverse                WorkflowStepName = "refresh_universe"
	WorkflowStepNameEvaluateRegime                 WorkflowStepName = "evaluate_regime"
	WorkflowStepNameBuildTradingReviewInputs       WorkflowStepName = "build_trading_review_inputs"
	WorkflowStepNameCreateTradingBatchJob          WorkflowStepName = "create_trading_batch_job"
	WorkflowStepNameSubmitTradingBatchJob          WorkflowStepName = "submit_trading_batch_job"
	WorkflowStepNameWaitForTradingAsyncResults     WorkflowStepName = "wait_for_trading_async_results"
	WorkflowStepNamePollAndReconcileTradingResults WorkflowStepName = "poll_and_reconcile_trading_batch_results"
	WorkflowStepNameValidateTradingAIOutputs       WorkflowStepName = "validate_trading_ai_outputs"
	WorkflowStepNameApproveTradeCandidates         WorkflowStepName = "approve_trade_candidates"
	WorkflowStepNamePersistTradingReview           WorkflowStepName = "persist_trading_review"
	WorkflowStepNamePublishTradingRunSummary       WorkflowStepName = "publish_trading_run_summary"
)

type WorkflowStepStatus string

const (
	WorkflowStepStatusPending         WorkflowStepStatus = "pending"
	WorkflowStepStatusRunning         WorkflowStepStatus = "running"
	WorkflowStepStatusWaitingExternal WorkflowStepStatus = "waiting_external"
	WorkflowStepStatusCompleted       WorkflowStepStatus = "completed"
	WorkflowStepStatusFailed          WorkflowStepStatus = "failed"
	WorkflowStepStatusSkipped         WorkflowStepStatus = "skipped"
)

type AIBatchJobType string

const (
	AIBatchJobTypeInvestingReviewBatch AIBatchJobType = "investing_review_batch"
	AIBatchJobTypeThesisUpdateBatch    AIBatchJobType = "thesis_update_batch"
	AIBatchJobTypeChangeDetectionBatch AIBatchJobType = "change_detection_batch"
	AIBatchJobTypeEvidenceSummaryBatch AIBatchJobType = "evidence_summary_batch"
	AIBatchJobTypeTradingReviewBatch   AIBatchJobType = "trading_review_batch"
)

type AIBatchJobStatus string

const (
	AIBatchJobStatusCreated            AIBatchJobStatus = "created"
	AIBatchJobStatusSubmitted          AIBatchJobStatus = "submitted"
	AIBatchJobStatusRunning            AIBatchJobStatus = "running"
	AIBatchJobStatusPartiallyCompleted AIBatchJobStatus = "partially_completed"
	AIBatchJobStatusCompleted          AIBatchJobStatus = "completed"
	AIBatchJobStatusFailed             AIBatchJobStatus = "failed"
	AIBatchJobStatusCancelled          AIBatchJobStatus = "cancelled"
	AIBatchJobStatusTimedOut           AIBatchJobStatus = "timed_out"
)

type AIBatchItemType string

const (
	AIBatchItemTypeCompanyReview          AIBatchItemType = "company_review"
	AIBatchItemTypeThesisUpdate           AIBatchItemType = "thesis_update"
	AIBatchItemTypeChangeSummary          AIBatchItemType = "change_summary"
	AIBatchItemTypeEvidenceSummary        AIBatchItemType = "evidence_summary"
	AIBatchItemTypeTradingCandidateReview AIBatchItemType = "trading_candidate_review"
)

type AIBatchItemStatus string

const (
	AIBatchItemStatusPending       AIBatchItemStatus = "pending"
	AIBatchItemStatusSubmitted     AIBatchItemStatus = "submitted"
	AIBatchItemStatusProcessing    AIBatchItemStatus = "processing"
	AIBatchItemStatusCompleted     AIBatchItemStatus = "completed"
	AIBatchItemStatusFailed        AIBatchItemStatus = "failed"
	AIBatchItemStatusInvalidOutput AIBatchItemStatus = "invalid_output"
	AIBatchItemStatusSkipped       AIBatchItemStatus = "skipped"
)

type ValidationStatus string

const (
	ValidationStatusNotValidated ValidationStatus = "not_validated"
	ValidationStatusValid        ValidationStatus = "valid"
	ValidationStatusInvalid      ValidationStatus = "invalid"
)

type RecommendedTrancheStyle string

const (
	RecommendedTrancheStyleStart  RecommendedTrancheStyle = "start"
	RecommendedTrancheStyleAdd    RecommendedTrancheStyle = "add"
	RecommendedTrancheStylePause  RecommendedTrancheStyle = "pause"
	RecommendedTrancheStyleReduce RecommendedTrancheStyle = "reduce"
	RecommendedTrancheStyleExit   RecommendedTrancheStyle = "exit"
)

type HardGateName string

const (
	HardGateNameGovernanceRedFlagAbsence HardGateName = "governance_red_flag_absence"
)

type SubScoreName string

const (
	SubScoreNameLiquidity                              SubScoreName = "liquidity"
	SubScoreNameDataQualityCompleteness                SubScoreName = "data_quality_completeness"
	SubScoreNameBasicInvestabilitySuitability          SubScoreName = "basic_investability_suitability"
	SubScoreNameListingOperatingHistorySufficiency     SubScoreName = "listing_operating_history_sufficiency"
	SubScoreNameRevenueGrowthStrength                  SubScoreName = "revenue_growth_strength"
	SubScoreNameRevenueGrowthConsistency               SubScoreName = "revenue_growth_consistency"
	SubScoreNameRecent12QAccelerationDeterioration     SubScoreName = "recent_12q_acceleration_deterioration"
	SubScoreNameEvidenceOfExpandingDemand              SubScoreName = "evidence_of_expanding_demand"
	SubScoreNameOperatingMarginQualityTrend            SubScoreName = "operating_margin_quality_trend"
	SubScoreNameProfitGrowthStrength                   SubScoreName = "profit_growth_strength"
	SubScoreNameCashConversionQuality                  SubScoreName = "cash_conversion_quality"
	SubScoreNameRecentOperatingLeverageMarginDirection SubScoreName = "recent_operating_leverage_margin_direction"
	SubScoreNameROCEROICQuality                        SubScoreName = "roce_roic_quality"
	SubScoreNameBalanceSheetStrength                   SubScoreName = "balance_sheet_strength"
	SubScoreNameWorkingCapitalEfficiency               SubScoreName = "working_capital_efficiency"
	SubScoreNameDilutionCapitalAllocationDiscipline    SubScoreName = "dilution_capital_allocation_discipline"
	SubScoreNameDemandTailwindStrength                 SubScoreName = "demand_tailwind_strength"
	SubScoreNameIndustryEconomicsQuality               SubScoreName = "industry_economics_quality"
	SubScoreNamePolicyFormalizationSupport             SubScoreName = "policy_formalization_support"
	SubScoreNameCyclicalityRisk                        SubScoreName = "cyclicality_risk"
	SubScoreNameMarketOpportunitySize                  SubScoreName = "market_opportunity_size"
	SubScoreNameShareGainPotential                     SubScoreName = "share_gain_potential"
	SubScoreNameExpansionOptionality                   SubScoreName = "expansion_optionality"
	SubScoreNameCompetitivePositioningStrength         SubScoreName = "competitive_positioning_strength"
	SubScoreNameCapitalAllocationQuality               SubScoreName = "capital_allocation_quality"
	SubScoreNameExecutionConsistency                   SubScoreName = "execution_consistency"
	SubScoreNameShareholderAlignmentTrustworthiness    SubScoreName = "shareholder_alignment_trustworthiness"
	SubScoreNameDisclosureQuality                      SubScoreName = "disclosure_quality"
	SubScoreNameRelativeStrength                       SubScoreName = "relative_strength"
	SubScoreNameTrendQuality                           SubScoreName = "trend_quality"
	SubScoreNameDrawdownResilienceBehavior             SubScoreName = "drawdown_resilience_behavior"
	SubScoreNameReactionToResultsNews                  SubScoreName = "reaction_to_results_news"
	SubScoreNameHistoricalValuationAttractiveness      SubScoreName = "historical_valuation_attractiveness"
	SubScoreNameValuationSupportVsCurrentQuality       SubScoreName = "valuation_support_vs_current_quality"
	SubScoreNameOvervaluationRisk                      SubScoreName = "overvaluation_risk"
	SubScoreNameEntryTimingSuitability                 SubScoreName = "entry_timing_suitability"
)

var validBookTypes = map[BookType]struct{}{
	BookTypeInvesting: {},
	BookTypeTrading:   {},
}

var validInvestingModes = map[InvestingMode]struct{}{
	InvestingModeEarlyHunter:         {},
	InvestingModeBalanced:            {},
	InvestingModeConfirmedCompounder: {},
}

var validReviewPeriodTypes = map[ReviewPeriodType]struct{}{
	ReviewPeriodTypeMonthly:     {},
	ReviewPeriodTypeQuarterly:   {},
	ReviewPeriodTypeEventDriven: {},
	ReviewPeriodTypeManual:      {},
}

var validReviewStatuses = map[ReviewStatus]struct{}{
	ReviewStatusDraft:      {},
	ReviewStatusFinal:      {},
	ReviewStatusSuperseded: {},
}

var validReviewLifecycleStates = map[ReviewLifecycleState]struct{}{
	ReviewLifecycleStatePendingInput:           {},
	ReviewLifecycleStatePendingAI:              {},
	ReviewLifecycleStateAICompletedUnvalidated: {},
	ReviewLifecycleStateValidationFailed:       {},
	ReviewLifecycleStateAIValidated:            {},
	ReviewLifecycleStateFinalized:              {},
	ReviewLifecycleStateSuperseded:             {},
}

var validReviewerTypes = map[ReviewerType]struct{}{
	ReviewerTypeAI:     {},
	ReviewerTypeHuman:  {},
	ReviewerTypeHybrid: {},
}

var validWatchlistBuckets = map[WatchlistBucket]struct{}{
	WatchlistBucketResearch:   {},
	WatchlistBucketWatch:      {},
	WatchlistBucketBuyReady:   {},
	WatchlistBucketHold:       {},
	WatchlistBucketExitReview: {},
}

var validInvestingActionTypes = map[InvestingActionType]struct{}{
	InvestingActionTypeBuy:    {},
	InvestingActionTypeWatch:  {},
	InvestingActionTypeHold:   {},
	InvestingActionTypeTrim:   {},
	InvestingActionTypeSell:   {},
	InvestingActionTypeReject: {},
}

var validSectionNames = map[SectionName]struct{}{
	SectionNameInvestability:                      {},
	SectionNameBusinessTraction:                   {},
	SectionNameProfitConversion:                   {},
	SectionNameCapitalEfficiencyFinancialStrength: {},
	SectionNameStructuralSectorAttractiveness:     {},
	SectionNameRunwayIndustryPositioning:          {},
	SectionNameManagementGovernance:               {},
	SectionNameMarketConfirmation:                 {},
	SectionNameValuationEntryAttractiveness:       {},
}

var validSectionActionCaps = map[SectionActionCap]struct{}{
	SectionActionCapCannotBuy:      {},
	SectionActionCapWatchOnly:      {},
	SectionActionCapExitReviewOnly: {},
	SectionActionCapNone:           {},
	"":                             {},
}

var validTrendDirections = map[TrendDirection]struct{}{
	TrendDirectionImproving: {},
	TrendDirectionStable:    {},
	TrendDirectionWeakening: {},
	TrendDirectionMixed:     {},
}

var validMetricBases = map[MetricBasis]struct{}{
	MetricBasisQuant:  {},
	MetricBasisText:   {},
	MetricBasisHybrid: {},
}

var validEvidenceStrengths = map[EvidenceStrength]struct{}{
	EvidenceStrengthLow:    {},
	EvidenceStrengthMedium: {},
	EvidenceStrengthHigh:   {},
	"":                     {},
}

var validEvidenceSourceTypes = map[EvidenceSourceType]struct{}{
	EvidenceSourceTypeAnnualReport:         {},
	EvidenceSourceTypeConcall:              {},
	EvidenceSourceTypeInvestorPresentation: {},
	EvidenceSourceTypeExchangeFiling:       {},
	EvidenceSourceTypeFinancialData:        {},
	EvidenceSourceTypePriceData:            {},
	EvidenceSourceTypeManualNote:           {},
}

var validEvidenceDirections = map[EvidenceDirection]struct{}{
	EvidenceDirectionPositive: {},
	EvidenceDirectionNegative: {},
	EvidenceDirectionNeutral:  {},
	"":                        {},
}

var validThesisStatuses = map[ThesisStatus]struct{}{
	ThesisStatusActive:      {},
	ThesisStatusUnderReview: {},
	ThesisStatusBroken:      {},
	ThesisStatusArchived:    {},
}

var validPositionRoles = map[PositionRole]struct{}{
	PositionRoleStarter:       {},
	PositionRoleBuilding:      {},
	PositionRoleCore:          {},
	PositionRoleTrimCandidate: {},
	PositionRoleExitCandidate: {},
	"":                        {},
}

var validWorkflowRunTypes = map[WorkflowRunType]struct{}{
	WorkflowRunTypeMonthlyScan:      {},
	WorkflowRunTypeQuarterlyRefresh: {},
	WorkflowRunTypeEventRefresh:     {},
	WorkflowRunTypeManual:           {},
}

var validWorkflowRunStatuses = map[WorkflowRunStatus]struct{}{
	WorkflowRunStatusCreated:            {},
	WorkflowRunStatusRunning:            {},
	WorkflowRunStatusWaitingExternal:    {},
	WorkflowRunStatusPartiallyCompleted: {},
	WorkflowRunStatusCompleted:          {},
	WorkflowRunStatusFailed:             {},
	WorkflowRunStatusCancelled:          {},
}

var validWorkflowStepNames = map[WorkflowStepName]struct{}{
	WorkflowStepNameScanUniverse:                   {},
	WorkflowStepNameApplyHardFilters:               {},
	WorkflowStepNameBuildReviewInputs:              {},
	WorkflowStepNameCreatePendingReviewRecords:     {},
	WorkflowStepNameCreateBatchJob:                 {},
	WorkflowStepNameSubmitBatchJob:                 {},
	WorkflowStepNameWaitForAsyncResults:            {},
	WorkflowStepNamePollAndReconcileBatchResults:   {},
	WorkflowStepNameValidateAIOutputs:              {},
	WorkflowStepNameMaterializeFinalReviews:        {},
	WorkflowStepNameEvaluateThesisAndChange:        {},
	WorkflowStepNameMapActions:                     {},
	WorkflowStepNameAssignBuckets:                  {},
	WorkflowStepNameBuildCapitalCandidates:         {},
	WorkflowStepNameAllocateCapital:                {},
	WorkflowStepNamePersistOutputs:                 {},
	WorkflowStepNamePublishRunSummary:              {},
	WorkflowStepNameRefreshUniverse:                {},
	WorkflowStepNameEvaluateRegime:                 {},
	WorkflowStepNameBuildTradingReviewInputs:       {},
	WorkflowStepNameCreateTradingBatchJob:          {},
	WorkflowStepNameSubmitTradingBatchJob:          {},
	WorkflowStepNameWaitForTradingAsyncResults:     {},
	WorkflowStepNamePollAndReconcileTradingResults: {},
	WorkflowStepNameValidateTradingAIOutputs:       {},
	WorkflowStepNameApproveTradeCandidates:         {},
	WorkflowStepNamePersistTradingReview:           {},
	WorkflowStepNamePublishTradingRunSummary:       {},
}

var validWorkflowStepStatuses = map[WorkflowStepStatus]struct{}{
	WorkflowStepStatusPending:         {},
	WorkflowStepStatusRunning:         {},
	WorkflowStepStatusWaitingExternal: {},
	WorkflowStepStatusCompleted:       {},
	WorkflowStepStatusFailed:          {},
	WorkflowStepStatusSkipped:         {},
}

var validAIBatchJobTypes = map[AIBatchJobType]struct{}{
	AIBatchJobTypeInvestingReviewBatch: {},
	AIBatchJobTypeThesisUpdateBatch:    {},
	AIBatchJobTypeChangeDetectionBatch: {},
	AIBatchJobTypeEvidenceSummaryBatch: {},
	AIBatchJobTypeTradingReviewBatch:   {},
}

var validAIBatchJobStatuses = map[AIBatchJobStatus]struct{}{
	AIBatchJobStatusCreated:            {},
	AIBatchJobStatusSubmitted:          {},
	AIBatchJobStatusRunning:            {},
	AIBatchJobStatusPartiallyCompleted: {},
	AIBatchJobStatusCompleted:          {},
	AIBatchJobStatusFailed:             {},
	AIBatchJobStatusCancelled:          {},
	AIBatchJobStatusTimedOut:           {},
}

var validAIBatchItemTypes = map[AIBatchItemType]struct{}{
	AIBatchItemTypeCompanyReview:          {},
	AIBatchItemTypeThesisUpdate:           {},
	AIBatchItemTypeChangeSummary:          {},
	AIBatchItemTypeEvidenceSummary:        {},
	AIBatchItemTypeTradingCandidateReview: {},
}

var validAIBatchItemStatuses = map[AIBatchItemStatus]struct{}{
	AIBatchItemStatusPending:       {},
	AIBatchItemStatusSubmitted:     {},
	AIBatchItemStatusProcessing:    {},
	AIBatchItemStatusCompleted:     {},
	AIBatchItemStatusFailed:        {},
	AIBatchItemStatusInvalidOutput: {},
	AIBatchItemStatusSkipped:       {},
}

var validValidationStatuses = map[ValidationStatus]struct{}{
	ValidationStatusNotValidated: {},
	ValidationStatusValid:        {},
	ValidationStatusInvalid:      {},
}

var validRecommendedTrancheStyles = map[RecommendedTrancheStyle]struct{}{
	RecommendedTrancheStyleStart:  {},
	RecommendedTrancheStyleAdd:    {},
	RecommendedTrancheStylePause:  {},
	RecommendedTrancheStyleReduce: {},
	RecommendedTrancheStyleExit:   {},
	"":                            {},
}

var validHardGateNames = map[HardGateName]struct{}{
	HardGateNameGovernanceRedFlagAbsence: {},
}

var validSubScoreNames = map[SubScoreName]struct{}{
	SubScoreNameLiquidity:                              {},
	SubScoreNameDataQualityCompleteness:                {},
	SubScoreNameBasicInvestabilitySuitability:          {},
	SubScoreNameListingOperatingHistorySufficiency:     {},
	SubScoreNameRevenueGrowthStrength:                  {},
	SubScoreNameRevenueGrowthConsistency:               {},
	SubScoreNameRecent12QAccelerationDeterioration:     {},
	SubScoreNameEvidenceOfExpandingDemand:              {},
	SubScoreNameOperatingMarginQualityTrend:            {},
	SubScoreNameProfitGrowthStrength:                   {},
	SubScoreNameCashConversionQuality:                  {},
	SubScoreNameRecentOperatingLeverageMarginDirection: {},
	SubScoreNameROCEROICQuality:                        {},
	SubScoreNameBalanceSheetStrength:                   {},
	SubScoreNameWorkingCapitalEfficiency:               {},
	SubScoreNameDilutionCapitalAllocationDiscipline:    {},
	SubScoreNameDemandTailwindStrength:                 {},
	SubScoreNameIndustryEconomicsQuality:               {},
	SubScoreNamePolicyFormalizationSupport:             {},
	SubScoreNameCyclicalityRisk:                        {},
	SubScoreNameMarketOpportunitySize:                  {},
	SubScoreNameShareGainPotential:                     {},
	SubScoreNameExpansionOptionality:                   {},
	SubScoreNameCompetitivePositioningStrength:         {},
	SubScoreNameCapitalAllocationQuality:               {},
	SubScoreNameExecutionConsistency:                   {},
	SubScoreNameShareholderAlignmentTrustworthiness:    {},
	SubScoreNameDisclosureQuality:                      {},
	SubScoreNameRelativeStrength:                       {},
	SubScoreNameTrendQuality:                           {},
	SubScoreNameDrawdownResilienceBehavior:             {},
	SubScoreNameReactionToResultsNews:                  {},
	SubScoreNameHistoricalValuationAttractiveness:      {},
	SubScoreNameValuationSupportVsCurrentQuality:       {},
	SubScoreNameOvervaluationRisk:                      {},
	SubScoreNameEntryTimingSuitability:                 {},
}

func (value BookType) IsValid() bool             { return isAllowed(value, validBookTypes) }
func (value InvestingMode) IsValid() bool        { return isAllowed(value, validInvestingModes) }
func (value ReviewPeriodType) IsValid() bool     { return isAllowed(value, validReviewPeriodTypes) }
func (value ReviewStatus) IsValid() bool         { return isAllowed(value, validReviewStatuses) }
func (value ReviewLifecycleState) IsValid() bool { return isAllowed(value, validReviewLifecycleStates) }
func (value ReviewerType) IsValid() bool         { return isAllowed(value, validReviewerTypes) }
func (value WatchlistBucket) IsValid() bool      { return isAllowed(value, validWatchlistBuckets) }
func (value InvestingActionType) IsValid() bool  { return isAllowed(value, validInvestingActionTypes) }
func (value SectionName) IsValid() bool          { return isAllowed(value, validSectionNames) }
func (value SectionActionCap) IsValid() bool     { return isAllowed(value, validSectionActionCaps) }
func (value TrendDirection) IsValid() bool       { return isAllowed(value, validTrendDirections) }
func (value MetricBasis) IsValid() bool          { return isAllowed(value, validMetricBases) }
func (value EvidenceStrength) IsValid() bool     { return isAllowed(value, validEvidenceStrengths) }
func (value EvidenceSourceType) IsValid() bool   { return isAllowed(value, validEvidenceSourceTypes) }
func (value EvidenceDirection) IsValid() bool    { return isAllowed(value, validEvidenceDirections) }
func (value ThesisStatus) IsValid() bool         { return isAllowed(value, validThesisStatuses) }
func (value PositionRole) IsValid() bool         { return isAllowed(value, validPositionRoles) }
func (value WorkflowRunType) IsValid() bool      { return isAllowed(value, validWorkflowRunTypes) }
func (value WorkflowRunStatus) IsValid() bool    { return isAllowed(value, validWorkflowRunStatuses) }
func (value WorkflowStepName) IsValid() bool     { return isAllowed(value, validWorkflowStepNames) }
func (value WorkflowStepStatus) IsValid() bool   { return isAllowed(value, validWorkflowStepStatuses) }
func (value AIBatchJobType) IsValid() bool       { return isAllowed(value, validAIBatchJobTypes) }
func (value AIBatchJobStatus) IsValid() bool     { return isAllowed(value, validAIBatchJobStatuses) }
func (value AIBatchItemType) IsValid() bool      { return isAllowed(value, validAIBatchItemTypes) }
func (value AIBatchItemStatus) IsValid() bool    { return isAllowed(value, validAIBatchItemStatuses) }
func (value ValidationStatus) IsValid() bool     { return isAllowed(value, validValidationStatuses) }
func (value RecommendedTrancheStyle) IsValid() bool {
	return isAllowed(value, validRecommendedTrancheStyles)
}
func (value HardGateName) IsValid() bool { return isAllowed(value, validHardGateNames) }
func (value SubScoreName) IsValid() bool { return isAllowed(value, validSubScoreNames) }
