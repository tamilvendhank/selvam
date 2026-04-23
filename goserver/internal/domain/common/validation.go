package common

import (
	"fmt"
	"math"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const floatTolerance = 0.0001

func isAllowed[T comparable](value T, valid map[T]struct{}) bool {
	_, ok := valid[value]
	return ok
}

func RequireObjectID(field string, id primitive.ObjectID) error {
	if id.IsZero() {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

func RequireString(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

func RequireTime(field string, value time.Time) error {
	if value.IsZero() {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

func ValidateSchemaVersion(field string, value int) error {
	if value <= 0 {
		return fmt.Errorf("%s must be greater than zero", field)
	}
	return nil
}

func ValidateNonNegativeFloat(field string, value float64) error {
	if value < 0 {
		return fmt.Errorf("%s must be zero or greater", field)
	}
	return nil
}

func ValidateNonNegativeInt(field string, value int) error {
	if value < 0 {
		return fmt.Errorf("%s must be zero or greater", field)
	}
	return nil
}

func ValidatePositiveInt(field string, value int) error {
	if value <= 0 {
		return fmt.Errorf("%s must be greater than zero", field)
	}
	return nil
}

// ValidateScore expects a human-facing score in the closed interval [1, 10].
func ValidateScore(field string, value float64) error {
	if value < 1 || value > 10 {
		return fmt.Errorf("%s must be between 1 and 10", field)
	}
	return nil
}

// ValidateComputedScore expects a computed aggregate score in the closed interval [0, 10].
func ValidateComputedScore(field string, value float64) error {
	if value < 0 || value > 10 {
		return fmt.Errorf("%s must be between 0 and 10", field)
	}
	return nil
}

func ValidateUnitInterval(field string, value float64) error {
	if value < 0 || value > 1 {
		return fmt.Errorf("%s must be between 0 and 1", field)
	}
	return nil
}

func ValidatePercentage(field string, value float64) error {
	if value < 0 || value > 100 {
		return fmt.Errorf("%s must be between 0 and 100", field)
	}
	return nil
}

func ValidateTimestampOrder(earlierField string, earlier time.Time, laterField string, later time.Time) error {
	if later.Before(earlier) {
		return fmt.Errorf("%s cannot be before %s", laterField, earlierField)
	}
	return nil
}

func ValidateOptionalTimestampOrder(earlierField string, earlier time.Time, laterField string, later *time.Time) error {
	if later == nil {
		return nil
	}
	return ValidateTimestampOrder(earlierField, earlier, laterField, later.UTC())
}

func ValidateStringSlice(field string, values []string) error {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s cannot contain blank values", field)
		}
	}
	return nil
}

func NearlyEqual(left, right float64) bool {
	return math.Abs(left-right) <= floatTolerance
}

func NormalizeWeightedScore(rawScore, weight float64) float64 {
	return math.Round((rawScore*weight/100)*10000) / 10000
}
