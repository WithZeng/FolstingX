package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	DB     DBConfig     `mapstructure:"database"`
	Auth   AuthConfig   `mapstructure:"auth"`
	Log    LogConfig    `mapstructure:"log"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DBConfig struct {
	Type string `mapstructure:"type"`
	DSN  string `mapstructure:"dsn"`
}

type AuthConfig struct {
	JWTSecret string `mapstructure:"jwt_secret"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{Host: "0.0.0.0", Port: 8080, Mode: "release"},
		DB:     DBConfig{Type: "sqlite", DSN: "./data/folstingx.db"},
		Auth:   AuthConfig{JWTSecret: "change-me"},
		Log:    LogConfig{Level: "info", File: "./logs/app.log"},
	}
}

func LoadConfig(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetEnvPrefix("FOLSTINGX")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	def := DefaultConfig()
	v.SetDefault("server.host", def.Server.Host)
	v.SetDefault("server.port", def.Server.Port)
	v.SetDefault("server.mode", def.Server.Mode)
	v.SetDefault("database.type", def.DB.Type)
	v.SetDefault("database.dsn", def.DB.DSN)
	v.SetDefault("auth.jwt_secret", def.Auth.JWTSecret)
	v.SetDefault("log.level", def.Log.Level)
	v.SetDefault("log.file", def.Log.File)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config failed: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}
	return &cfg, nil
}
