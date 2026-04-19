package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"goserver/internal/domain"
	"goserver/internal/repository"
	"goserver/internal/shared"

	"go.mongodb.org/mongo-driver/bson"
)

type ProceduresService struct {
	repo *repository.ProceduresRepository
}

func NewProceduresService(repo *repository.ProceduresRepository) *ProceduresService {
	return &ProceduresService{repo: repo}
}

func (service *ProceduresService) GetProceduresForList(ctx context.Context) ([]map[string]any, error) {
	procedures, err := service.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]any, 0, len(procedures))
	for _, procedure := range procedures {
		result = append(result, procedureViewModel(procedure))
	}

	return result, nil
}

func (service *ProceduresService) GetProcedureDetails(ctx context.Context, id string) (map[string]any, error) {
	procedure, err := service.repo.GetByID(ctx, id)
	if err != nil || procedure == nil {
		return nil, err
	}

	return procedureViewModel(procedure), nil
}

func (service *ProceduresService) CreateProcedureDefinition(ctx context.Context, name string, rawSteps []map[string]any) (map[string]any, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, fmt.Errorf("procedure name is required")
	}

	steps, err := normalizeProcedureSteps(rawSteps)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	procedure, err := service.repo.Create(ctx, &domain.Procedure{
		Name:      trimmedName,
		Steps:     steps,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, err
	}

	return procedureViewModel(procedure), nil
}

func (service *ProceduresService) UpdateProcedureDefinition(ctx context.Context, id, name string, rawSteps []map[string]any) (map[string]any, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, fmt.Errorf("procedure name is required")
	}

	existingProcedure, err := service.repo.GetByID(ctx, id)
	if err != nil || existingProcedure == nil {
		return nil, err
	}

	steps, err := normalizeProcedureSteps(rawSteps)
	if err != nil {
		return nil, err
	}

	procedure, err := service.repo.Update(ctx, id, bson.M{
		"name":  trimmedName,
		"steps": steps,
	})
	if err != nil {
		return nil, err
	}

	return procedureViewModel(procedure), nil
}

func normalizeProcedureSteps(rawSteps []map[string]any) ([]domain.ProcedureStep, error) {
	steps := make([]domain.ProcedureStep, 0, len(rawSteps))
	for _, rawStep := range rawSteps {
		prompt, _ := rawStep["prompt"].(string)
		prompt = strings.TrimSpace(prompt)
		if prompt == "" {
			continue
		}

		model, _ := rawStep["model"].(string)
		resolvedModel := shared.NormalizeModelName(firstNonEmpty(model, shared.DefaultModel))
		reasoning, _ := rawStep["reasoningEffort"].(string)
		steps = append(steps, domain.ProcedureStep{
			Prompt:          prompt,
			Model:           resolvedModel,
			ReasoningEffort: shared.NormalizeReasoningEffort(resolvedModel, reasoning),
		})
	}

	if len(steps) == 0 {
		return nil, fmt.Errorf("please add at least one procedure step with a prompt")
	}

	for index := range steps {
		steps[index].ID = fmt.Sprintf("step-%d", index+1)
		steps[index].StepNumber = index + 1
	}

	return steps, nil
}

func procedureViewModel(procedure *domain.Procedure) map[string]any {
	if procedure == nil {
		return nil
	}

	return map[string]any{
		"id":             procedure.ID,
		"name":           procedure.Name,
		"steps":          shared.NormalizeJSONValue(procedure.Steps),
		"createdAt":      procedure.CreatedAt,
		"updatedAt":      procedure.UpdatedAt,
		"stepCount":      len(procedure.Steps),
		"createdAtLabel": shared.FormatDateLabel(&procedure.CreatedAt, "Unknown"),
		"updatedAtLabel": shared.FormatDateLabel(&procedure.UpdatedAt, "Unknown"),
	}
}
