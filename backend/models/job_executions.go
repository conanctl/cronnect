package models

import "time"

type JobExecution struct {
	ID           string     `gorm:"primaryKey" json:"id"`
	JobID        string     `gorm:"index;not null" json:"job_id"`
	StartedAt    time.Time  `gorm:"autoCreateTime" json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	Status       string     `gorm:"size:20;not null" json:"status"`
	ResponseCode int        `json:"response_code"`
}
