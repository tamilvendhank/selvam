package service

import (
	"context"
	"fmt"

	"goserver/internal/platform/domain"
	"goserver/internal/platform/ports"
)

type DefaultCompanyService struct {
	companyRepository  ports.CompanyRepository
	reviewRepository   ports.CompanyReviewRepository
	thesisRepository   ports.ThesisRepository
	positionRepository ports.PositionRepository
}

func NewCompanyService(
	companyRepository ports.CompanyRepository,
	reviewRepository ports.CompanyReviewRepository,
	thesisRepository ports.ThesisRepository,
	positionRepository ports.PositionRepository,
) *DefaultCompanyService {
	return &DefaultCompanyService{
		companyRepository:  companyRepository,
		reviewRepository:   reviewRepository,
		thesisRepository:   thesisRepository,
		positionRepository: positionRepository,
	}
}

func (service *DefaultCompanyService) ListCompanies(ctx context.Context, filter ports.CompanyListFilter) ([]*domain.Company, error) {
	return service.companyRepository.List(ctx, filter)
}

func (service *DefaultCompanyService) GetCompany(ctx context.Context, id string) (*domain.Company, error) {
	company, err := service.companyRepository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, ErrNotFound
	}

	return company, nil
}

func (service *DefaultCompanyService) ListCompanyReviews(ctx context.Context, companyID string, filter ports.CompanyReviewListFilter) ([]*domain.CompanyReview, error) {
	filter.CompanyID = companyID
	return service.reviewRepository.List(ctx, filter)
}

func (service *DefaultCompanyService) GetCompanyThesis(ctx context.Context, companyID string) (*domain.InvestmentThesis, error) {
	thesis, err := service.thesisRepository.GetActiveByCompanyID(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if thesis == nil {
		return nil, ErrNotFound
	}

	return thesis, nil
}

func (service *DefaultCompanyService) GetHistorySummary(ctx context.Context, companyID string, bookType domain.BookType) (map[string]any, error) {
	company, err := service.companyRepository.GetByID(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, ErrNotFound
	}

	reviews, err := service.reviewRepository.List(ctx, ports.CompanyReviewListFilter{
		CompanyID: companyID,
		BookType:  bookType,
		Limit:     12,
	})
	if err != nil {
		return nil, err
	}

	thesis, err := service.thesisRepository.GetActiveByCompanyID(ctx, companyID)
	if err != nil {
		return nil, err
	}

	position, err := service.positionRepository.GetByCompanyAndBook(ctx, companyID, bookType)
	if err != nil {
		return nil, err
	}

	var latestReview *domain.CompanyReview
	if len(reviews) > 0 {
		latestReview = reviews[0]
	}

	summary := map[string]any{
		"companyId":    companyID,
		"symbol":       company.Symbol,
		"bookType":     bookType,
		"reviewCount":  len(reviews),
		"latestReview": latestReview,
		"thesis":       thesis,
		"position":     position,
		"hasThesis":    thesis != nil,
		"isOwned":      position != nil && position.Quantity > 0,
	}
	if latestReview != nil {
		summary["latestAction"] = latestReview.FinalActionAfterReview
		summary["latestBucket"] = latestReview.FinalBucketAfterReview
		summary["latestScore"] = latestReview.WeightedTotalScore
	}

	return summary, nil
}

func ensureReviewExists(review *domain.CompanyReview) error {
	if review == nil {
		return fmt.Errorf("review is required")
	}

	return nil
}
