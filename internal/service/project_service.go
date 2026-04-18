package service

import (
	"context"
	"database/sql"
	"errors"
	"llm-usage-tracker/internal/store"
	"strings"
)

// Not in the same package as store, so we still have to create a new type
// to add custom funcs to.
type ProjectService struct {
	repo *store.ProjectRepo
}

func NewProjectService(repo *store.ProjectRepo) *ProjectService {
	return &ProjectService{repo: repo}
}


func isUniqueConstraintError(err error) bool {
    return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

var (
    ErrDuplicateName   = errors.New("duplicate name")
    ErrInvalidName     = errors.New("name cannot be empty")
    ErrInvalidBudget   = errors.New("budget must be positive")
    ErrNotFound        = errors.New("project not found")
)

func (s *ProjectService) CreateProject(ctx context.Context, name string, budget int) (*store.Project, error) {
	if name == "" {
		return nil, ErrInvalidName
	}
	if budget <= 0 {
		return nil, ErrInvalidBudget
	}

	project := store.Project{
		Name: name,
		Budget: int64(budget),
	}

	err := s.repo.Create(ctx, &project)
	if err != nil {
		if isUniqueConstraintError(err) {
            return nil, ErrDuplicateName
        }
		return nil, err
	}

	return &project, nil
}

func (s *ProjectService) ListProjects() ([]store.Project, error) {
	projects, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	return projects, nil
}

func (s *ProjectService) GetProjectByID(ctx context.Context, id int64) (*store.Project, error) {
	project, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return project, nil
}

func (s *ProjectService) UpdateProject(ctx context.Context, id int64, name *string, budget *int) (*store.Project, error) {
	project, err := s.GetProjectByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if name != nil {
		if *name == "" {
			return nil, ErrInvalidName
		}
		project.Name = *name
	}

	if budget != nil {
		if *budget <= 0 {
			return nil, ErrInvalidBudget
		}
		project.Budget = int64(*budget)
	}

	err = s.repo.Update(ctx, project)
	if err != nil {
		if isUniqueConstraintError(err) {
            return nil, ErrDuplicateName
        }
		return nil, err
	}

	return project, nil
}

func (s *ProjectService) DeleteProject(ctx context.Context, id int64) error {
	// First check if it exists so we can return ErrNotFound if it doesn't
	_, err := s.GetProjectByID(ctx, id)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, id)
}