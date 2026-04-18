package store

import (
	"context"
	"database/sql"
)

type ModelRepo struct {
	db *sql.DB
}

func NewModelRepo(db *sql.DB) *ModelRepo {
	return &ModelRepo{db: db}
}

func (r *ModelRepo) Create(ctx context.Context, m *Model) error {
	res, err := r.db.ExecContext(
		ctx,
		"INSERT INTO models(name, input_per_million_cents, output_per_million_cents) VALUES (?, ?, ?)",
		m.Name, m.InputPerMillionCents, m.OutputPerMillionCents,
	)
	if err != nil {
		return err
	}

	id, _ := res.LastInsertId()
	m.ID = id
	return nil
}

func (r *ModelRepo) List(ctx context.Context) ([]Model, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, name, input_per_million_cents, output_per_million_cents, created_at FROM models ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Model
	for rows.Next() {
		var m Model
		if err := rows.Scan(&m.ID, &m.Name, &m.InputPerMillionCents, &m.OutputPerMillionCents, &m.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, m)
	}

	return result, nil
}

func (r *ModelRepo) GetByID(ctx context.Context, id int64) (*Model, error) {
	var m Model
	err := r.db.QueryRowContext(
		ctx,
		"SELECT id, name, input_per_million_cents, output_per_million_cents, created_at FROM models WHERE id = ?",
		id,
	).Scan(&m.ID, &m.Name, &m.InputPerMillionCents, &m.OutputPerMillionCents, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *ModelRepo) Update(ctx context.Context, m *Model) error {
	_, err := r.db.ExecContext(
		ctx,
		"UPDATE models SET name = ?, input_per_million_cents = ?, output_per_million_cents = ? WHERE id = ?",
		m.Name, m.InputPerMillionCents, m.OutputPerMillionCents, m.ID,
	)
	return err
}

func (r *ModelRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(
		ctx,
		"DELETE FROM models WHERE id = ?",
		id,
	)
	return err
}
