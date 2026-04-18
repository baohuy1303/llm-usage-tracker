package service

import (
	"context"
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

var ErrDuplicateName = errors.New("project name already exists")

func (s *ProjectService) CreateProject(ctx context.Context, name string, budget int) (*store.Project, error) {
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}
	if budget <= 0 {
		return nil, errors.New("budget must be positive")
	}

	project := store.Project{
		Name: name,
		Budget: int64(budget),
	}

	err := s.repo.Create(ctx, &project)
	if err != nil {
		if isUniqueConstraintError(err) {
            return nil, errors.New("project name already exists")
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
		return nil, err
	}
	return project, nil
}