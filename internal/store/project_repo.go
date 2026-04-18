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


func (r *ProjectRepo) Create(ctx context.Context, p *Project) error {
    res, err := r.db.ExecContext(
        ctx,
        "INSERT INTO projects(name, budget) VALUES (?, ?)",
        p.Name, p.Budget,
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
    rows, err := r.db.Query("SELECT id, name, budget, created_at FROM projects")
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
        rows.Scan(&p.ID, &p.Name, &p.Budget, &p.CreatedAt)
        result = append(result, p)
    }

    return result, nil
}

func (r *ProjectRepo) GetByID(ctx context.Context, id int64) (*Project, error) {
	var p Project
	err := r.db.QueryRowContext(
		ctx,
		"SELECT id, name, budget, created_at FROM projects WHERE id = ?",
		id,
	).Scan(&p.ID, &p.Name, &p.Budget, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectRepo) Update(ctx context.Context, p *Project) error {
	_, err := r.db.ExecContext(
		ctx,
		"UPDATE projects SET name = ?, budget = ? WHERE id = ?",
		p.Name, p.Budget, p.ID,
	)
	return err
}

func (r *ProjectRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(
		ctx,
		"DELETE FROM projects WHERE id = ?",
		id,
	)
	return err
}
