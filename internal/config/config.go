package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         string `yaml:"env" env-default:"local"`
	StoragePath string `yaml:"storage_path" env-required:"true"`

	TokenSecret string `yaml:"token_secret" env-default:"super-secret-key-change-me"`

	HTTPServer `yaml:"http_server"`
	Database   `yaml:"database"`
	Redis      `yaml:"redis"`
	MinIO      `yaml:"minio"`
}

type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"localhost:8080"`
	Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

type Database struct {
	DSN string `yaml:"dsn" env-required:"true"` // DSN - Data Source Name (строка подключения)
}

type Redis struct {
	Address string `yaml:"address" env-default:"localhost:6379"`
}

type MinIO struct {
	Endpoint        string `yaml:"endpoint" env-default:"localhost:9000"`
	AccessKeyID     string `yaml:"access_key_id" env-default:"minio_user"`
	SecretAccessKey string `yaml:"secret_access_key" env-default:"minio_password"`
	Bucket          string `yaml:"bucket" env-default:"images"`
	UseSSL          bool   `yaml:"use_ssl" env-default:"false"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")

	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %s", err)
	}

	return &cfg
}
