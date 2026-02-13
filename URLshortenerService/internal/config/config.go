package config

import (
	"URLshortener/internal/lib/api/response"
	"errors"
	"flag"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"os"
	"time"
)

type Config struct {
	Env        string `yaml:"env" env-default:"local"`
	DBDriver   string `yaml:"db_driver"`
	ConnString string `yaml:"conn_string"`
	Secret     string `yaml:"secret"`
	HTTPServer `yaml:"http_server"`
}

type DBInitData struct {
	DB_NAME     string `validate:"required,min=1,max=64"`
	DB_USERNAME string `validate:"required,alphanum,min=3,max=32"`
	DB_PASSWORD string `validate:"required"`
	DB_HOST     string `validate:"required,hostname|ip"`
	DB_PORT     string `validate:"required,number,min=1,max=65535"`
}

type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"localhost:8080"`
	Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
	User        string        `yaml:"user" env-required:"true"`
	Password    string        `yaml:"password" env-required:"true" env:"HTTP_SERVER_PASSWORD"`
}

func MustLoad() *Config {
	var configPath string

	flag.StringVar(&configPath, "config", "", "path to config file")
	flag.Parse()

	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
	}

	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}

	// проверка на наличие файла
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %v", err)
	}

	urls_db := DBInitData{
		DB_NAME:     os.Getenv("DB_URLS_NAME"),
		DB_USERNAME: os.Getenv("DB_URLS_USERNAME"),
		DB_PASSWORD: os.Getenv("DB_URLS_PASSWORD"),
		DB_HOST:     os.Getenv("DB_URLS_HOST"),
		DB_PORT:     os.Getenv("DB_URLS_PORT"),
	}

	if err := validator.New().Struct(urls_db); err != nil {
		var validateErr validator.ValidationErrors
		if !errors.As(err, &validateErr) {
			log.Fatalf("cannot to validate urls_db init data")
		}

		log.Fatalf(response.ValidateEnvVar(validateErr))
	}

	cfg.ConnString = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		urls_db.DB_USERNAME,
		urls_db.DB_PASSWORD,
		urls_db.DB_HOST,
		"5432",
		urls_db.DB_NAME,
	)

	secretKey := os.Getenv("SECRET_KEY")
	if secretKey == "" {
		log.Fatalf("env SECRET_KEY is required")
	}

	cfg.Secret = secretKey

	return &cfg
}
