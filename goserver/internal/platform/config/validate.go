package config

import (
	"fmt"
	"strings"

	"goserver/internal/platform/domain"
)

func (config AppConfig) Validate() error {
	if strings.TrimSpace(config.SchemaVersion) == "" {
		return fmt.Errorf("schemaVersion is required")
	}
	if config.Server.Port <= 0 {
		return fmt.Errorf("server.port must be greater than zero")
	}
	if strings.TrimSpace(config.Mongo.URI) == "" {
		return fmt.Errorf("mongo.uri is required")
	}
	if strings.TrimSpace(config.Mongo.Database) == "" {
		return fmt.Errorf("mongo.database is required")
	}
	if err := validateCollectionNames(config.Mongo.Collections); err != nil {
		return err
	}
	if strings.TrimSpace(config.Global.DefaultTimezone) == "" {
		return fmt.Errorf("global.defaultTimezone is required")
	}
	if err := validateInvestingConfig(config.Investing); err != nil {
		return err
	}
	if err := validateTradingConfig(config.Trading); err != nil {
		return err
	}
	if config.UI.DefaultPageSize <= 0 {
		return fmt.Errorf("ui.defaultPageSize must be greater than zero")
	}
	if config.UI.MaxPageSize < config.UI.DefaultPageSize {
		return fmt.Errorf("ui.maxPageSize must be greater than or equal to ui.defaultPageSize")
	}

	return nil
}

func validateCollectionNames(collections CollectionConfig) error {
	values := map[string]string{
		"companies":             collections.Companies,
		"companyReviews":        collections.CompanyReviews,
		"investmentTheses":      collections.InvestmentTheses,
		"workflowRuns":          collections.WorkflowRuns,
		"configSnapshots":       collections.ConfigSnapshots,
		"capitalAllocationRuns": collections.CapitalAllocationRuns,
		"manualOverrides":       collections.ManualOverrides,
		"currentPositions":      collections.CurrentPositions,
		"aiBatchJobs":           collections.AIBatchJobs,
		"aiBatchIterations":     collections.AIBatchIterations,
	}

	for name, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("mongo.collections.%s is required", name)
		}
	}

	return nil
}

func validateInvestingConfig(config InvestingConfig) error {
	if !domain.IsValidInvestingMode(domain.InvestingMode(config.DefaultMode)) {
		return fmt.Errorf("investing.defaultMode is invalid")
	}

	sectionWeights := make(map[string]float64, len(config.SectionWeights))
	var totalSectionWeights float64
	for _, item := range config.SectionWeights {
		if !domain.IsValidInvestingSectionName(item.Name) {
			return fmt.Errorf("invalid investing section weight name %q", item.Name)
		}
		if err := domain.ValidatePercentRange("investing section weight", item.Weight); err != nil {
			return err
		}
		if _, exists := sectionWeights[item.Name]; exists {
			return fmt.Errorf("duplicate investing section weight %q", item.Name)
		}
		sectionWeights[item.Name] = item.Weight
		totalSectionWeights += item.Weight
	}
	if domain.NormalizeScore(totalSectionWeights) != 100 {
		return fmt.Errorf("investing section weights must total 100")
	}
	if len(config.SectionWeights) != len(domain.InvestingSectionsInOrder) {
		return fmt.Errorf("investing section weights must include all defined sections")
	}

	seenSubScoreSections := make(map[string]struct{}, len(config.SubScoreWeights))
	for _, section := range config.SubScoreWeights {
		if !domain.IsValidInvestingSectionName(section.SectionName) {
			return fmt.Errorf("invalid sub-score weight section %q", section.SectionName)
		}
		if _, exists := seenSubScoreSections[section.SectionName]; exists {
			return fmt.Errorf("duplicate sub-score section %q", section.SectionName)
		}
		seenSubScoreSections[section.SectionName] = struct{}{}
		var total float64
		for _, subScore := range section.SubScores {
			if !domain.IsValidInvestingSubScore(section.SectionName, subScore.Name) {
				return fmt.Errorf("invalid sub-score %q for section %q", subScore.Name, section.SectionName)
			}
			if err := domain.ValidatePercentRange("investing sub-score weight", subScore.Weight); err != nil {
				return err
			}
			total += subScore.Weight
		}
		if domain.NormalizeScore(total) != 100 {
			return fmt.Errorf("sub-score weights for section %q must total 100", section.SectionName)
		}
	}
	if len(seenSubScoreSections) != len(domain.InvestingSectionsInOrder) {
		return fmt.Errorf("all investing sections must define sub-score weights")
	}

	requiredBuckets := map[string]struct{}{
		string(domain.WatchlistBucketResearch):   {},
		string(domain.WatchlistBucketWatch):      {},
		string(domain.WatchlistBucketBuyReady):   {},
		string(domain.WatchlistBucketHold):       {},
		string(domain.WatchlistBucketExitReview): {},
	}
	for _, bucket := range config.WatchlistBuckets {
		if !domain.IsValidBucket(domain.WatchlistBucket(bucket)) {
			return fmt.Errorf("invalid investing watchlist bucket %q", bucket)
		}
		delete(requiredBuckets, bucket)
	}
	if len(requiredBuckets) != 0 {
		return fmt.Errorf("investing.watchlistBuckets is missing required buckets")
	}

	for _, cadence := range config.ReviewCadenceByBucket {
		if !domain.IsValidBucket(domain.WatchlistBucket(cadence.Bucket)) {
			return fmt.Errorf("invalid cadence bucket %q", cadence.Bucket)
		}
		if cadence.ReviewEveryDays <= 0 {
			return fmt.Errorf("review cadence must be greater than zero days")
		}
	}

	if err := domain.ValidatePercentRange("min meaningful target pct", config.PositionSizing.MinMeaningfulTargetPct); err != nil {
		return err
	}
	if err := domain.ValidatePercentRange("max position cap pct", config.PositionSizing.MaxPositionCapPct); err != nil {
		return err
	}
	if config.PositionSizing.MinMeaningfulTargetPct > config.PositionSizing.MaxPositionCapPct {
		return fmt.Errorf("min meaningful target pct cannot exceed max position cap pct")
	}
	if err := validatePortfolioSplit(config.Allocation.PortfolioTargetSplit); err != nil {
		return err
	}
	if config.Allocation.DefaultTrancheCount <= 0 {
		return fmt.Errorf("allocation.defaultTrancheCount must be greater than zero")
	}

	for _, coreSection := range config.CoreSections {
		if !domain.IsValidInvestingSectionName(coreSection) {
			return fmt.Errorf("invalid core section %q", coreSection)
		}
	}

	if !domain.IsValidActionType(domain.ActionType(config.ValuationRules.ExtremeOvervaluationAction)) &&
		!domain.IsValidSectionActionCap(domain.SectionActionCap(config.ValuationRules.ExtremeOvervaluationAction)) {
		return fmt.Errorf("valuationRules.extremeOvervaluationAction must be a valid action or section action cap")
	}

	return nil
}

func validateTradingConfig(config TradingConfig) error {
	if err := domain.ValidatePercentRange("trading risk per trade pct", config.RiskPerTradePct); err != nil {
		return err
	}
	if config.MaxConcurrentPositions <= 0 {
		return fmt.Errorf("trading.maxConcurrentPositions must be greater than zero")
	}
	if strings.TrimSpace(config.StopStyle) == "" {
		return fmt.Errorf("trading.stopStyle is required")
	}
	if err := domain.ValidatePercentRange("trading drawdown kill switch pct", config.DrawdownKillSwitch.StopOpeningNewTradesDrawdownPct); err != nil {
		return err
	}
	if config.DrawdownKillSwitch.CooldownDays <= 0 {
		return fmt.Errorf("trading cooldownDays must be greater than zero")
	}

	return nil
}

func validatePortfolioSplit(split PortfolioSplit) error {
	if err := domain.ValidatePercentRange("investing portfolio target split", split.InvestingBookPct); err != nil {
		return err
	}
	if err := domain.ValidatePercentRange("trading portfolio target split", split.TradingBookPct); err != nil {
		return err
	}
	if err := domain.ValidatePercentRange("liquid reserve target split", split.LiquidReservePct); err != nil {
		return err
	}
	if domain.NormalizeScore(split.InvestingBookPct+split.TradingBookPct+split.LiquidReservePct) != 100 {
		return fmt.Errorf("portfolio target split must total 100")
	}

	return nil
}
