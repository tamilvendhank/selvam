package mongo

import (
	companypkg "goserver/internal/domain/company"
	reviewpkg "goserver/internal/domain/review"
	legacydomain "goserver/internal/platform/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type companyDocument struct {
	companypkg.Company `bson:",inline"`
}

type companyReviewDocument struct {
	reviewpkg.CompanyReview `bson:",inline"`
}

type jobReconciliationLogDocument struct {
	ObjectID                          primitive.ObjectID `bson:"_id,omitempty"`
	legacydomain.JobReconciliationLog `bson:",inline"`
}
