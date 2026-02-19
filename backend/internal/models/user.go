package models

import "time"

type UserRole string

const (
	RoleSuperAdmin UserRole = "super_admin"
	RoleAdmin      UserRole = "admin"
	RoleUser       UserRole = "user"
)

type User struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Username       string    `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash   string    `gorm:"size:255;not null" json:"-"`
	Role           UserRole  `gorm:"size:20;default:user;index" json:"role"`
	APIKey         string    `gorm:"size:128;uniqueIndex" json:"api_key"`
	BandwidthLimit int64     `gorm:"default:0" json:"bandwidth_limit"`
	TrafficLimit   int64     `gorm:"default:0" json:"traffic_limit"`
	TrafficUsed    int64     `gorm:"default:0" json:"traffic_used"`
	IsActive       bool      `gorm:"default:true" json:"is_active"`
	ExpireAt       time.Time `json:"expire_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}
