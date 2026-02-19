package models

import "time"

type SystemLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Level     string    `gorm:"size:20;index" json:"level"`
	Module    string    `gorm:"size:50;index" json:"module"`
	Message   string    `gorm:"type:text" json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

func (SystemLog) TableName() string {
	return "system_logs"
}
