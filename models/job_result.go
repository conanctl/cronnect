package models

import "time"

type JobResult struct {
	ExecutionID  string    `json:"execution_id"`
	Status       string    `json:"status"`
	ResponseCode int       `json:"response_code,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CompletedAt  time.Time `json:"completed_at"`
}
