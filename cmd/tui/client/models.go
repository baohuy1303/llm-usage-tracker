package client

import (
	"context"
	"strconv"
	"time"
)

// Model mirrors the wire shape from /models. Pricing is integer cents per
// million tokens (raw API representation; we don't dollar-ize it because the
// number is already small and human-readable).
type Model struct {
	ID                    int64     `json:"id"`
	Name                  string    `json:"name"`
	InputPerMillionCents  int64     `json:"input_per_million_cents"`
	OutputPerMillionCents int64     `json:"output_per_million_cents"`
	CreatedAt             time.Time `json:"created_at"`
}

type ModelInput struct {
	Name                  string `json:"name,omitempty"`
	InputPerMillionCents  *int64 `json:"input_per_million_cents,omitempty"`
	OutputPerMillionCents *int64 `json:"output_per_million_cents,omitempty"`
}

func (c *Client) ListModels(ctx context.Context) ([]Model, error) {
	var out []Model
	if err := c.do(ctx, "GET", "/models", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetModel(ctx context.Context, id int64) (*Model, error) {
	var out Model
	if err := c.do(ctx, "GET", "/models/"+itoa(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateModel(ctx context.Context, in ModelInput) (*Model, error) {
	var out Model
	if err := c.do(ctx, "POST", "/models", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateModel(ctx context.Context, id int64, in ModelInput) (*Model, error) {
	var out Model
	if err := c.do(ctx, "PATCH", "/models/"+itoa(id), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteModel(ctx context.Context, id int64) error {
	return c.do(ctx, "DELETE", "/models/"+itoa(id), nil, nil)
}

// itoa is shared by the project/model/usage files for path construction.
func itoa(n int64) string { return strconv.FormatInt(n, 10) }
