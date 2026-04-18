package store

import "time"

type Project struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Budget    int64     `json:"budget"`
	CreatedAt time.Time `json:"created_at"`
}