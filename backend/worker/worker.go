package worker

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/conan-flynn/cronnect/database"
	"github.com/conan-flynn/cronnect/models"
	"github.com/conan-flynn/cronnect/queue"
	"github.com/google/uuid"
)

type Worker struct {
	ID           string
	queueService *queue.QueueService
	httpClient   *http.Client
}

func NewWorker() *Worker {
	workerID := fmt.Sprintf("worker-%s", uuid.NewString()[:8])
	
	return &Worker{
		ID:           workerID,
		queueService: queue.NewQueueService(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}


func (w *Worker) Start() {
	log.Printf("Starting worker %s", w.ID)
	w.queueService.ConsumeJobs(w.ID, w.processJob)
}


func (w *Worker) processJob(payload *models.JobPayload) *models.JobResult {
	log.Printf("Worker %s: Executing job %s", w.ID, payload.Name)


	var execution models.JobExecution
	database.DB.First(&execution, "id = ?", payload.ExecutionID)
	execution.Status = "running"
	database.DB.Save(&execution)

	result := &models.JobResult{
		ExecutionID: payload.ExecutionID,
		CompletedAt: time.Now(),
	}


	req, err := http.NewRequest(payload.Method, payload.URL, nil)
	if err != nil {
		log.Printf("Worker %s: Failed to create request for job %s: %v", w.ID, payload.Name, err)
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}


	for key, value := range payload.Headers {
		req.Header.Set(key, value)
	}


	resp, err := w.httpClient.Do(req)
	if err != nil {
		log.Printf("Worker %s: Request failed for job %s: %v", w.ID, payload.Name, err)
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("HTTP request failed: %v", err)
		return result
	}
	defer resp.Body.Close()


	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Worker %s: Failed to read response body for job %s: %v", w.ID, payload.Name, err)
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("Failed to read response: %v", err)
		return result
	}

	result.ResponseCode = resp.StatusCode


	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Status = "success"
		log.Printf("Worker %s: Job %s completed successfully with status %d", w.ID, payload.Name, resp.StatusCode)
		log.Printf("Worker %s: Response body: %s", w.ID, string(body))
	} else {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("HTTP status %d: %s", resp.StatusCode, string(body))
		log.Printf("Worker %s: Job %s failed with status %d: %s", w.ID, payload.Name, resp.StatusCode, string(body))
	}

	return result
}


func StartMultipleWorkers(count int) {
	for i := 0; i < count; i++ {
		worker := NewWorker()
		go worker.Start()
		

		time.Sleep(100 * time.Millisecond)
	}
}
