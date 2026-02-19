package database

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/folstingx/server/config"
	"github.com/folstingx/server/internal/models"
	"github.com/folstingx/server/pkg/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(cfg *config.Config) error {
	if cfg.DB.Type != "sqlite" {
		return fmt.Errorf("unsupported db type: %s", cfg.DB.Type)
	}

	dsn := cfg.DB.DSN
	if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil {
		return err
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	if err := db.AutoMigrate(&models.User{}, &models.Node{}, &models.SystemLog{}, &models.ForwardRule{}, &models.TrafficStat{}, &models.Tunnel{}, &models.ChainTunnel{}, &models.Forward{}, &models.ForwardPort{}); err != nil {
		return err
	}

	DB = db
	return ensureDefaultAdmin()
}

func ensureDefaultAdmin() error {
	var count int64
	if err := DB.Model(&models.User{}).Where("username = ?", "admin").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hash, err := utils.HashPassword("admin123")
	if err != nil {
		return err
	}

	admin := &models.User{
		Username:     "admin",
		PasswordHash: hash,
		Role:         models.RoleSuperAdmin,
		IsActive:     true,
		ExpireAt:     time.Now().AddDate(20, 0, 0),
	}
	return DB.Create(admin).Error
}
