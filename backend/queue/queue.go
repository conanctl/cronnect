package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/conan-flynn/cronnect/database"
	"github.com/conan-flynn/cronnect/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	JobQueue      = "cronnect:jobs"
	ResultQueue   = "cronnect:results"
	RetryQueue    = "cronnect:retry"
	DeadQueue     = "cronnect:dead"
	DefaultMaxRetries = 3
)

type QueueService struct {
	client *redis.Client
	ctx    context.Context
}

func NewQueueService() *QueueService {
	return &QueueService{
		client: database.RedisClient,
		ctx:    context.Background(),
	}
}


func (qs *QueueService) PublishJob(job *models.Job) error {
	pendingKey := fmt.Sprintf("pending:%s", job.ID)
	
	exists, err := qs.client.Exists(qs.ctx, pendingKey).Result()
	if err != nil {
		log.Printf("Error checking pending job: %v", err)
	} else if exists > 0 {
		log.Printf("Job %s already has pending execution, skipping", job.Name)
		return nil
	}

	executionID := uuid.NewString()
	

	execution := models.JobExecution{
		ID:        executionID,
		JobID:     job.ID,
		StartedAt: time.Now(),
		Status:    "queued",
	}
	
	database.DB.Create(&execution)

	payload := models.JobPayload{
		JobID:       job.ID,
		Name:        job.Name,
		URL:         job.URL,
		Method:      job.Method,
		ExecutionID: executionID,
		ScheduledAt: time.Now(),
		MaxRetries:  DefaultMaxRetries,
		RetryCount:  0,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal job payload: %w", err)
	}

	qs.client.Set(qs.ctx, pendingKey, executionID, 10*time.Minute)

	err = qs.client.LPush(qs.ctx, JobQueue, payloadJSON).Err()
	if err != nil {
		qs.client.Del(qs.ctx, pendingKey)
		execution.Status = "failed"
		database.DB.Save(&execution)
		return fmt.Errorf("failed to publish job to queue: %w", err)
	}

	log.Printf("Published job %s (execution %s) to queue", job.Name, executionID)
	return nil
}


func (qs *QueueService) ConsumeJobs(workerID string, processFn func(*models.JobPayload) *models.JobResult) {
	log.Printf("Worker %s started consuming jobs from queue", workerID)
	
	for {

		result, err := qs.client.BRPop(qs.ctx, 5*time.Second, JobQueue).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			log.Printf("Worker %s: Error consuming job: %v", workerID, err)
			time.Sleep(1 * time.Second)
			continue
		}

		if len(result) < 2 {
			log.Printf("Worker %s: Invalid result format", workerID)
			continue
		}

		jobData := result[1]
		var payload models.JobPayload
		
		if err := json.Unmarshal([]byte(jobData), &payload); err != nil {
			log.Printf("Worker %s: Failed to unmarshal job payload: %v", workerID, err)
			continue
		}

		log.Printf("Worker %s: Processing job %s (execution %s)", workerID, payload.Name, payload.ExecutionID)
		jobResult := processFn(&payload)


		if err := qs.handleJobResult(&payload, jobResult); err != nil {
			log.Printf("Worker %s: Failed to handle job result: %v", workerID, err)
		}
	}
}


func (qs *QueueService) handleJobResult(payload *models.JobPayload, result *models.JobResult) error {

	var execution models.JobExecution
	if err := database.DB.First(&execution, "id = ?", result.ExecutionID).Error; err != nil {
		return fmt.Errorf("failed to find execution record: %w", err)
	}

	execution.Status = result.Status
	execution.ResponseCode = result.ResponseCode
	execution.FinishedAt = &result.CompletedAt

	pendingKey := fmt.Sprintf("pending:%s", payload.JobID)

	if result.Status == "failed" && payload.RetryCount < payload.MaxRetries {
		payload.RetryCount++
		if err := qs.requeueForRetry(payload); err != nil {
			log.Printf("Failed to requeue job for retry: %v", err)
			qs.moveToDeadQueue(payload, result.ErrorMessage)
			qs.client.Del(qs.ctx, pendingKey)
		} else {
			execution.Status = "retry"
			log.Printf("Job %s queued for retry (attempt %d/%d)", payload.Name, payload.RetryCount, payload.MaxRetries)
		}
	} else {
		qs.client.Del(qs.ctx, pendingKey)
		if result.Status == "failed" {
			qs.moveToDeadQueue(payload, result.ErrorMessage)
			log.Printf("Job %s moved to dead letter queue after %d failed attempts", payload.Name, payload.RetryCount)
		}
	}

	return database.DB.Save(&execution).Error
}


func (qs *QueueService) requeueForRetry(payload *models.JobPayload) error {

	delay := time.Duration(payload.RetryCount*payload.RetryCount) * time.Minute
	
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}


	score := float64(time.Now().Add(delay).Unix())
	return qs.client.ZAdd(qs.ctx, RetryQueue, redis.Z{
		Score:  score,
		Member: payloadJSON,
	}).Err()
}


func (qs *QueueService) moveToDeadQueue(payload *models.JobPayload, errorMsg string) {
	deadJob := map[string]interface{}{
		"payload":       payload,
		"error_message": errorMsg,
		"failed_at":     time.Now(),
	}

	deadJobJSON, _ := json.Marshal(deadJob)
	qs.client.LPush(qs.ctx, DeadQueue, deadJobJSON)
}


func (qs *QueueService) ProcessRetryQueue() {
	for {
		now := float64(time.Now().Unix())
		

		results, err := qs.client.ZRangeByScoreWithScores(qs.ctx, RetryQueue, &redis.ZRangeBy{
			Min: "0",
			Max: fmt.Sprintf("%.0f", now),
		}).Result()

		if err != nil {
			log.Printf("Error processing retry queue: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}

		for _, result := range results {
			jobData := result.Member.(string)
			

			qs.client.ZRem(qs.ctx, RetryQueue, jobData)
			

			qs.client.LPush(qs.ctx, JobQueue, jobData)
			
			log.Printf("Moved job from retry queue back to main queue")
		}

		time.Sleep(30 * time.Second)
	}
}
