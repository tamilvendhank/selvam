package mongo

import (
	"goserver/internal/platform/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func toCompanyDocument(company *domain.Company) *companyDocument {
	if company == nil {
		return nil
	}

	return &companyDocument{
		Company: *company,
	}
}

func fromCompanyDocument(document *companyDocument) *domain.Company {
	if document == nil {
		return nil
	}

	company := document.Company
	company.ID = document.ObjectID.Hex()
	return &company
}

func toCompanyReviewDocument(review *domain.CompanyReview) *companyReviewDocument {
	if review == nil {
		return nil
	}

	return &companyReviewDocument{CompanyReview: *review}
}

func fromCompanyReviewDocument(document *companyReviewDocument) *domain.CompanyReview {
	if document == nil {
		return nil
	}

	review := document.CompanyReview
	review.ID = document.ObjectID.Hex()
	return &review
}

func toInvestmentThesisDocument(thesis *domain.InvestmentThesis) *investmentThesisDocument {
	if thesis == nil {
		return nil
	}

	return &investmentThesisDocument{InvestmentThesis: *thesis}
}

func fromInvestmentThesisDocument(document *investmentThesisDocument) *domain.InvestmentThesis {
	if document == nil {
		return nil
	}

	thesis := document.InvestmentThesis
	thesis.ID = document.ObjectID.Hex()
	return &thesis
}

func toWorkflowRunDocument(run *domain.WorkflowRun) *workflowRunDocument {
	if run == nil {
		return nil
	}

	return &workflowRunDocument{WorkflowRun: *run}
}

func fromWorkflowRunDocument(document *workflowRunDocument) *domain.WorkflowRun {
	if document == nil {
		return nil
	}

	run := document.WorkflowRun
	run.ID = document.ObjectID.Hex()
	return &run
}

func toWorkflowStepRunDocument(run *domain.WorkflowStepRun) *workflowStepRunDocument {
	if run == nil {
		return nil
	}

	return &workflowStepRunDocument{WorkflowStepRun: *run}
}

func fromWorkflowStepRunDocument(document *workflowStepRunDocument) *domain.WorkflowStepRun {
	if document == nil {
		return nil
	}

	run := document.WorkflowStepRun
	run.ID = document.ObjectID.Hex()
	return &run
}

func toConfigSnapshotDocument(snapshot *domain.ConfigSnapshot) *configSnapshotDocument {
	if snapshot == nil {
		return nil
	}

	return &configSnapshotDocument{ConfigSnapshot: *snapshot}
}

func fromConfigSnapshotDocument(document *configSnapshotDocument) *domain.ConfigSnapshot {
	if document == nil {
		return nil
	}

	snapshot := document.ConfigSnapshot
	snapshot.ID = document.ObjectID.Hex()
	return &snapshot
}

func toCapitalAllocationRunDocument(run *domain.CapitalAllocationRun) *capitalAllocationRunDocument {
	if run == nil {
		return nil
	}

	return &capitalAllocationRunDocument{CapitalAllocationRun: *run}
}

func fromCapitalAllocationRunDocument(document *capitalAllocationRunDocument) *domain.CapitalAllocationRun {
	if document == nil {
		return nil
	}

	run := document.CapitalAllocationRun
	run.ID = document.ObjectID.Hex()
	return &run
}

func toManualOverrideDocument(override *domain.ManualOverride) *manualOverrideDocument {
	if override == nil {
		return nil
	}

	return &manualOverrideDocument{ManualOverride: *override}
}

func fromManualOverrideDocument(document *manualOverrideDocument) *domain.ManualOverride {
	if document == nil {
		return nil
	}

	override := document.ManualOverride
	override.ID = document.ObjectID.Hex()
	return &override
}

func toCurrentPositionDocument(position *domain.CurrentPosition) *currentPositionDocument {
	if position == nil {
		return nil
	}

	return &currentPositionDocument{CurrentPosition: *position}
}

func fromCurrentPositionDocument(document *currentPositionDocument) *domain.CurrentPosition {
	if document == nil {
		return nil
	}

	position := document.CurrentPosition
	position.ID = document.ObjectID.Hex()
	return &position
}

func toAIBatchJobDocument(job *domain.AIBatchJob) *aiBatchJobDocument {
	if job == nil {
		return nil
	}

	return &aiBatchJobDocument{AIBatchJob: *job}
}

func fromAIBatchJobDocument(document *aiBatchJobDocument) *domain.AIBatchJob {
	if document == nil {
		return nil
	}

	job := document.AIBatchJob
	job.ID = document.ObjectID.Hex()
	return &job
}

func toAIBatchItemDocument(item *domain.AIBatchItem) *aiBatchItemDocument {
	if item == nil {
		return nil
	}

	return &aiBatchItemDocument{AIBatchItem: *item}
}

func fromAIBatchItemDocument(document *aiBatchItemDocument) *domain.AIBatchItem {
	if document == nil {
		return nil
	}

	item := document.AIBatchItem
	item.ID = document.ObjectID.Hex()
	return &item
}

func toJobReconciliationLogDocument(log *domain.JobReconciliationLog) *jobReconciliationLogDocument {
	if log == nil {
		return nil
	}

	return &jobReconciliationLogDocument{JobReconciliationLog: *log}
}

func fromJobReconciliationLogDocument(document *jobReconciliationLogDocument) *domain.JobReconciliationLog {
	if document == nil {
		return nil
	}

	log := document.JobReconciliationLog
	log.ID = document.ObjectID.Hex()
	return &log
}

func newDocumentID() primitive.ObjectID {
	return primitive.NewObjectID()
}
