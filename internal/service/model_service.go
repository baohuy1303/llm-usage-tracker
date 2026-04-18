package service

import (
	"context"
	"database/sql"
	"errors"
	"llm-usage-tracker/internal/store"
)

var (
	ErrModelNotFound = errors.New("model not found")
	ErrInvalidPricing = errors.New("pricing cannot be negative")
)

type ModelService struct {
	repo *store.ModelRepo
}

func NewModelService(repo *store.ModelRepo) *ModelService {
	return &ModelService{repo: repo}
}

func (s *ModelService) CreateModel(ctx context.Context, name string, inputCents, outputCents int64) (*store.Model, error) {
	if name == "" {
		return nil, ErrInvalidName
	}
	if inputCents < 0 || outputCents < 0 {
		return nil, ErrInvalidPricing
	}

	m := store.Model{
		Name:                  name,
		InputPerMillionCents:  inputCents,
		OutputPerMillionCents: outputCents,
	}

	err := s.repo.Create(ctx, &m)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrDuplicateName
		}
		return nil, err
	}

	return &m, nil
}

func (s *ModelService) ListModels(ctx context.Context) ([]store.Model, error) {
	models, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return models, nil
}

func (s *ModelService) GetModelByID(ctx context.Context, id int64) (*store.Model, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrModelNotFound
		}
		return nil, err
	}
	return m, nil
}

func (s *ModelService) UpdateModel(ctx context.Context, id int64, name *string, inputCents, outputCents *int64) (*store.Model, error) {
	m, err := s.GetModelByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if name != nil {
		if *name == "" {
			return nil, ErrInvalidName
		}
		m.Name = *name
	}

	if inputCents != nil {
		if *inputCents < 0 {
			return nil, ErrInvalidPricing
		}
		m.InputPerMillionCents = *inputCents
	}

	if outputCents != nil {
		if *outputCents < 0 {
			return nil, ErrInvalidPricing
		}
		m.OutputPerMillionCents = *outputCents
	}

	err = s.repo.Update(ctx, m)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrDuplicateName
		}
		return nil, err
	}

	return m, nil
}

func (s *ModelService) DeleteModel(ctx context.Context, id int64) error {
	// Verify exists first
	_, err := s.GetModelByID(ctx, id)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, id)
}
