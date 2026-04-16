package service

import (
	"errors"
	"llm-usage-tracker/internal/store"
)

// Not in the same package as store, so we still have to create a new type
// to add custom funcs to.
type ProjectService struct {
	repo *store.ProjectRepo
}

func NewProjectService(repo *store.ProjectRepo) *ProjectService {
	return &ProjectService{repo: repo}
}

func (s *ProjectService) CreateProject(name string, budget int) error {
	if name == "" {
		return errors.New("name cannot be empty")
	}
	if budget <= 0 {
		return errors.New("budget must be positive")
	}

	project := store.Project{
		Name: name,
		Budget: int64(budget),
	}

	err := s.repo.Create(&project)
	if err != nil {
		return err
	}

	return nil
}

func (s *ProjectService) ListProjects() ([]store.Project, error) {
	projects, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	return projects, nil
}