package mongo

import (
	"goserver/internal/platform/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type companyDocument struct {
	ObjectID       primitive.ObjectID `bson:"_id,omitempty"`
	domain.Company `bson:",inline"`
}

type companyReviewDocument struct {
	ObjectID             primitive.ObjectID `bson:"_id,omitempty"`
	domain.CompanyReview `bson:",inline"`
}

type investmentThesisDocument struct {
	ObjectID                primitive.ObjectID `bson:"_id,omitempty"`
	domain.InvestmentThesis `bson:",inline"`
}

type workflowRunDocument struct {
	ObjectID           primitive.ObjectID `bson:"_id,omitempty"`
	domain.WorkflowRun `bson:",inline"`
}

type configSnapshotDocument struct {
	ObjectID              primitive.ObjectID `bson:"_id,omitempty"`
	domain.ConfigSnapshot `bson:",inline"`
}

type capitalAllocationRunDocument struct {
	ObjectID                    primitive.ObjectID `bson:"_id,omitempty"`
	domain.CapitalAllocationRun `bson:",inline"`
}

type manualOverrideDocument struct {
	ObjectID              primitive.ObjectID `bson:"_id,omitempty"`
	domain.ManualOverride `bson:",inline"`
}

type currentPositionDocument struct {
	ObjectID               primitive.ObjectID `bson:"_id,omitempty"`
	domain.CurrentPosition `bson:",inline"`
}
