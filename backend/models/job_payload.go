package models

import "time"


type JobPayload struct {
	JobID        string            `json:"job_id"`
	Name         string            `json:"name"`
	URL          string            `json:"url"`
	Method       string            `json:"method"`
	Headers      map[string]string `json:"headers,omitempty"`
	Body         string            `json:"body,omitempty"`
	ExecutionID  string            `json:"execution_id"`
	ScheduledAt  time.Time         `json:"scheduled_at"`
	MaxRetries   int               `json:"max_retries"`
	RetryCount   int               `json:"retry_count"`
}


