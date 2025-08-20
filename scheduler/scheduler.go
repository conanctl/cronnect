package scheduler

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/conan-flynn/cronnect/database"
	"github.com/conan-flynn/cronnect/models"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

var c *cron.Cron

func StartScheduler() {
	c = cron.New()
	loadJobsFromDB()
	c.Start()

	go func() {
		for {
			time.Sleep(10 * time.Second)
			loadJobsFromDB()
		}
	}()
}

func loadJobsFromDB() {
	var jobs []models.Job
	database.DB.Preload("Executions").Find(&jobs)

	c.Stop()
	c = cron.New()

	for _, job := range jobs {
		j := job
		_, err := c.AddFunc(j.Schedule, func() {
			ExecuteJob(&j)
		})
		if err != nil {
			log.Printf("Failed to schedule job %s: %v", j.Name, err)
		} else {
			log.Printf("Scheduled job: %s", j.Name)
		}
	}

	c.Start()
}

func ExecuteJob(job *models.Job) {
	log.Printf("Executing job: %s", job.Name)

	execution := models.JobExecution{
		ID:        uuid.NewString(),
		JobID:     job.ID,
		StartedAt: time.Now(),
		Status:    "running",
	}

	database.DB.Create(&execution)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(job.Method, job.URL, nil)
	if err != nil {
		log.Printf("Failed to create request for job %s: %v", job.Name, err)
		execution.Status = "failed"
		database.DB.Save(&execution)
		return
	}

	resp, err := client.Do(req)
	execution.FinishedAt = ptrTime(time.Now())
	if err != nil {
		log.Printf("Request failed for job %s: %v", job.Name, err)
		execution.Status = "failed"
	} else {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		execution.Status = "success"
		execution.ResponseCode = resp.StatusCode

		log.Printf("Job %s completed with status %d", job.Name, resp.StatusCode)
		log.Printf("Response body: %s", string(body))
	}

	database.DB.Save(&execution)
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
