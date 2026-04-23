package service

import (
	"context"
	"database/sql"
	"errors"
	"llm-usage-tracker/internal/store"
	"log/slog"
	"strings"
)

// Not in the same package as store, so we still have to create a new type
// to add custom funcs to.
type ProjectService struct {
	repo         *store.ProjectRepo
	usageService *UsageService
}

func NewProjectService(repo *store.ProjectRepo, usageService *UsageService) *ProjectService {
	return &ProjectService{repo: repo, usageService: usageService}
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

// ProjectWithBudget wraps a Project with current budget status (daily/monthly/total).
// BudgetStatus is omitted when Redis is unavailable or the project has no budgets set.
type ProjectWithBudget struct {
	store.Project
	BudgetStatus *BudgetStatus `json:"budget_status,omitempty"`
}

// validateBudget returns ErrInvalidBudget if the pointer is set and <= 0.
// Nil means "no budget enforcement" and is allowed.
func validateBudget(b *int64) error {
	if b != nil && *b <= 0 {
		return ErrInvalidBudget
	}
	return nil
}

func (s *ProjectService) CreateProject(ctx context.Context, name string, daily, monthly, total *int64) (*store.Project, error) {
	if name == "" {
		return nil, ErrInvalidName
	}
	if err := validateBudget(daily); err != nil {
		return nil, err
	}
	if err := validateBudget(monthly); err != nil {
		return nil, err
	}
	if err := validateBudget(total); err != nil {
		return nil, err
	}

	project := store.Project{
		Name:               name,
		DailyBudgetCents:   daily,
		MonthlyBudgetCents: monthly,
		TotalBudgetCents:   total,
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

// getProject is the internal existence check used by Update/Delete.
// Public callers should use GetProjectByID which also attaches BudgetStatus.
func (s *ProjectService) getProject(ctx context.Context, id int64) (*store.Project, error) {
	project, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return project, nil
}

func (s *ProjectService) GetProjectByID(ctx context.Context, id int64) (*ProjectWithBudget, error) {
	project, err := s.getProject(ctx, id)
	if err != nil {
		return nil, err
	}

	result := &ProjectWithBudget{Project: *project}

	status, err := s.usageService.ComputeBudgetStatus(ctx, project)
	if err != nil {
		slog.Warn("budget status computation failed", "err", err, "project_id", id)
	} else {
		result.BudgetStatus = status
	}

	return result, nil
}

func (s *ProjectService) UpdateProject(ctx context.Context, id int64, name *string, daily, monthly, total *int64) (*store.Project, error) {
	project, err := s.getProject(ctx, id)
	if err != nil {
		return nil, err
	}

	if name != nil {
		if *name == "" {
			return nil, ErrInvalidName
		}
		project.Name = *name
	}

	// Each budget is *int64: nil = leave unchanged, non-nil value = set.
	if daily != nil {
		if err := validateBudget(daily); err != nil {
			return nil, err
		}
		project.DailyBudgetCents = daily
	}
	if monthly != nil {
		if err := validateBudget(monthly); err != nil {
			return nil, err
		}
		project.MonthlyBudgetCents = monthly
	}
	if total != nil {
		if err := validateBudget(total); err != nil {
			return nil, err
		}
		project.TotalBudgetCents = total
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
	_, err := s.getProject(ctx, id)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, id)
}
