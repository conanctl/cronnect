package scheduler

import (
	"log"

	"github.com/conan-flynn/cronnect/database"
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

	if err := queueService.PublishJob(job); err != nil {
		log.Printf("Failed to publish job %s to queue: %v", job.Name, err)
	}
}
