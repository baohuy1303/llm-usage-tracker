package store

import(
	"database/sql"
	"context"
)

// Go doesn't let us add custome funcs to a type that's not defined in the same package
// (database/sql is from the stdlib, not our package)
// So we create a new type ProjectRepo to add custom query logic
type ProjectRepo struct {
	db *sql.DB
}

// Constructor function: pass in a pointer to the database connection
// and return the address of that to the caller (repo object)
func NewProjectRepo(db *sql.DB) *ProjectRepo {
	return &ProjectRepo{db: db}
}

// (r *ProjectRepo) is called a receiver
// Means this func belongs to this type (like defining a class then having a method with this. or self.)

// nullIntFromPtr converts a *int64 (nullable budget) into sql.NullInt64 for INSERT/UPDATE.
func nullIntFromPtr(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

// ptrFromNullInt converts sql.NullInt64 (from a SELECT) back to *int64 for the struct.
func ptrFromNullInt(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

func (r *ProjectRepo) Create(ctx context.Context, p *Project) error {
    res, err := r.db.ExecContext(
        ctx,
        "INSERT INTO projects(name, daily_budget_millicents, monthly_budget_millicents, total_budget_millicents) VALUES (?, ?, ?, ?)",
        p.Name,
        nullIntFromPtr(p.DailyBudgetMillicents),
        nullIntFromPtr(p.MonthlyBudgetMillicents),
        nullIntFromPtr(p.TotalBudgetMillicents),
    )
    if err != nil {
        return err
    }

    id, _ := res.LastInsertId()
    p.ID = id
    return nil
}

func (r *ProjectRepo) List() ([]Project, error) {
	// Open a connection to the database
    rows, err := r.db.Query("SELECT id, name, daily_budget_millicents, monthly_budget_millicents, total_budget_millicents, created_at FROM projects WHERE deleted_at IS NULL")
    if err != nil {
        return nil, err
    }

	// Close the connection when the function returns
	// Put after err check, because rows might be nil
    defer rows.Close()

    var result []Project
	// Loop through the rows and fill our result slice
    for rows.Next() {
        var p Project
        var daily, monthly, total sql.NullInt64
        rows.Scan(&p.ID, &p.Name, &daily, &monthly, &total, &p.CreatedAt)
        p.DailyBudgetMillicents = ptrFromNullInt(daily)
        p.MonthlyBudgetMillicents = ptrFromNullInt(monthly)
        p.TotalBudgetMillicents = ptrFromNullInt(total)
        result = append(result, p)
    }

    return result, nil
}

func (r *ProjectRepo) GetByID(ctx context.Context, id int64) (*Project, error) {
	var p Project
	var daily, monthly, total sql.NullInt64
	err := r.db.QueryRowContext(
		ctx,
		"SELECT id, name, daily_budget_millicents, monthly_budget_millicents, total_budget_millicents, created_at FROM projects WHERE id = ? AND deleted_at IS NULL",
		id,
	).Scan(&p.ID, &p.Name, &daily, &monthly, &total, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	p.DailyBudgetMillicents = ptrFromNullInt(daily)
	p.MonthlyBudgetMillicents = ptrFromNullInt(monthly)
	p.TotalBudgetMillicents = ptrFromNullInt(total)
	return &p, nil
}

func (r *ProjectRepo) Update(ctx context.Context, p *Project) error {
	_, err := r.db.ExecContext(
		ctx,
		"UPDATE projects SET name = ?, daily_budget_millicents = ?, monthly_budget_millicents = ?, total_budget_millicents = ? WHERE id = ? AND deleted_at IS NULL",
		p.Name,
		nullIntFromPtr(p.DailyBudgetMillicents),
		nullIntFromPtr(p.MonthlyBudgetMillicents),
		nullIntFromPtr(p.TotalBudgetMillicents),
		p.ID,
	)
	return err
}

func (r *ProjectRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(
		ctx,
		"UPDATE projects SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?",
		id,
	)
	return err
}
