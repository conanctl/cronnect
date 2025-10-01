package models

type Job struct {
	ID         string         `gorm:"primaryKey" json:"id"`
	UserID     string         `gorm:"not null;index" json:"user_id"`
	Name       string         `gorm:"size:100;not null" json:"name"`
	URL        string         `gorm:"not null" json:"url"`
	Method     string         `gorm:"size:10;default:GET" json:"method"`
	Schedule   string         `gorm:"size:100;not null" json:"schedule"`
	Status     string         `gorm:"size:20;default:active" json:"status"`
	User       User           `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Executions []JobExecution `gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE" json:"executions"`
}
