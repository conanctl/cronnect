package scheduler

import (
	"log"

	"github.com/conan-flynn/cronnect/database"
	"github.com/conan-flynn/cronnect/middleware"
	"github.com/conan-flynn/cronnect/models"
	"github.com/conan-flynn/cronnect/queue"
	"github.com/robfig/cron/v3"
)

var c *cron.Cron
var queueService *queue.QueueService

func StartScheduler() {
	queueService = queue.NewQueueService()
	c = cron.New()
	loadJobsFromDB()
	c.Start()
}

func ReloadJobs() {
	loadJobsFromDB()
}

func loadJobsFromDB() {
	var jobs []models.Job
	database.DB.Preload("Executions").Find(&jobs)

	c.Stop()
	c = cron.New()

	for _, job := range jobs {
		j := job
		_, err := c.AddFunc(j.Schedule, func() {
			ScheduleJob(&j)
		})
		if err != nil {
			log.Printf("Failed to schedule job %s: %v", j.Name, err)
		} else {
			log.Printf("Scheduled job: %s", j.Name)
		}
	}

	c.Start()
}

func ScheduleJob(job *models.Job) {
	log.Printf("Scheduling job: %s", job.Name)

	rateLimiter := middleware.NewRateLimiter()
	allowed, remaining, resetAt, err := rateLimiter.CheckRateLimit(job.UserID)
	if err != nil {
		log.Printf("Failed to check rate limit for job %s: %v", job.Name, err)
	} else if !allowed {
		log.Printf("Rate limit exceeded for user %s. Job %s skipped. Limit resets at %s. Remaining: %d", 
			job.UserID, job.Name, resetAt.Format("15:04:05"), remaining)
		return
	}

	// Record the ping
	if err := rateLimiter.RecordPing(job.UserID); err != nil {
		log.Printf("Failed to record ping for user %s: %v", job.UserID, err)
	}

	if err := queueService.PublishJob(job); err != nil {
		log.Printf("Failed to publish job %s to queue: %v", job.Name, err)
	}
}
