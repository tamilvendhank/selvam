package mongo

import (
	companypkg "goserver/internal/domain/company"
	reviewpkg "goserver/internal/domain/review"
	legacydomain "goserver/internal/platform/domain"
)

func toCompanyDocument(company *companypkg.Company) *companyDocument {
	if company == nil {
		return nil
	}

	document := companyDocument{Company: *company}
	return &document
}

func fromCompanyDocument(document *companyDocument) *companypkg.Company {
	if document == nil {
		return nil
	}

	company := document.Company
	return &company
}

func toCompanyReviewDocument(review *reviewpkg.CompanyReview) *companyReviewDocument {
	if review == nil {
		return nil
	}

	document := companyReviewDocument{CompanyReview: *review}
	return &document
}

func fromCompanyReviewDocument(document *companyReviewDocument) *reviewpkg.CompanyReview {
	if document == nil {
		return nil
	}

	review := document.CompanyReview
	return &review
}

func toJobReconciliationLogDocument(log *legacydomain.JobReconciliationLog) *jobReconciliationLogDocument {
	if log == nil {
		return nil
	}

	document := jobReconciliationLogDocument{JobReconciliationLog: *log}
	return &document
}

func fromJobReconciliationLogDocument(document *jobReconciliationLogDocument) *legacydomain.JobReconciliationLog {
	if document == nil {
		return nil
	}

	log := document.JobReconciliationLog
	log.ID = document.ObjectID.Hex()
	return &log
}
