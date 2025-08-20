package scheduler

import (
	"log"
	"time"

	"github.com/conan-flynn/cronnect/database"
	"github.com/conan-flynn/cronnect/models"
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
	// TODO: add actual HTTP request logic here
}
