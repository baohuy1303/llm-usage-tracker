package store

type Project struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Budget    int64  `json:"budget"`
	CreatedAt int64  `json:"created_at"`
}