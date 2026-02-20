package job

import "time"

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

type Job struct {
	ID           int64     `json:"id"`
	Source       string    `json:"source"`
	Symbol       string    `json:"symbol"`
	StartDate    time.Time `json:"startDate"`
	EndDate      time.Time `json:"endDate"`
	Status       Status    `json:"status"`
	Error        string    `json:"error,omitempty"`
	RecordsCount int64     `json:"recordsCount"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
