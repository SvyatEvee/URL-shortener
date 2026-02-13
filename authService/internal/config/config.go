package config

import (
	"errors"
	"flag"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"os"
	"sso/internal/lib/api"
	"time"
)

type Config struct {
	Env                       string        `yaml:"env" env-default:"local"`
	MainStorageDBDriver       string        `yaml:"mainStorage_db_driver"`
	MainStorageConnString     string        `yaml:"mainStorage_conn_string"`
	SessionsStorageDBDriver   string        `yaml:"sessionsStorage_db_driver"`
	SessionsStorageConnString string        `yaml:"sessionsStorage_conn_string"`
	Secret                    string        `yaml:"secret"`
	AccessTokenTTL            time.Duration `yaml:"access_token_ttl" env-required:"true"`
	RefreshTokenTTL           time.Duration `yaml:"refresh_token_ttl" env-required:"true"`
	GRPC                      GRPCConfig    `yaml:"grpc"`
	Gateway                   GatewayConfig `yaml:"gateway"`
	UrlService                UrlService    `yaml:"urlService"`
}

type DBInitData struct {
	DB_NAME     string `validate:"required,min=1,max=64"`
	DB_USERNAME string `validate:"required,alphanum,min=3,max=32"`
	DB_PASSWORD string `validate:"required"`
	DB_HOST     string `validate:"required,hostname|ip"`
	DB_PORT     string `validate:"required,number,min=1,max=65535"`
}

type GRPCConfig struct {
	Host    string        `yaml:"host"`
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

type GatewayConfig struct {
	Port        int           `yaml:"gateway_port"`
	Enabled     bool          `yaml:"enabled"`
	Timeout     time.Duration `yaml:"timeout"`
	IdleTimeout time.Duration `yaml:"idle_timeout"`
}

type UrlService struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

func MustLoad() *Config {
	configPath := fetchConfigPath()
	if configPath == "" {
		log.Fatal("config path is not set")
	}

	return MustLoadByPath(configPath)
}

func MustLoadByPath(configPath string) *Config {
	// проверка на наличие файла
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %v", err)
	}

	main_db := DBInitData{
		DB_NAME:     os.Getenv("DB_USERS_NAME"),
		DB_USERNAME: os.Getenv("DB_USERS_USERNAME"),
		DB_PASSWORD: os.Getenv("DB_USERS_PASSWORD"),
		DB_HOST:     os.Getenv("DB_USERS_HOST"),
		DB_PORT:     os.Getenv("DB_USERS_PORT"),
	}

	sessions_db := DBInitData{
		DB_NAME:     os.Getenv("DB_SESSIONS_NAME"),
		DB_USERNAME: os.Getenv("DB_SESSIONS_USERNAME"),
		DB_PASSWORD: os.Getenv("DB_SESSIONS_PASSWORD"),
		DB_HOST:     os.Getenv("DB_SESSIONS_HOST"),
		DB_PORT:     os.Getenv("DB_SESSIONS_PORT"),
	}

	if err := validator.New().Struct(main_db); err != nil {
		var validateErr validator.ValidationErrors
		if !errors.As(err, &validateErr) {
			log.Fatalf("cannot to validate main_db init data")
		}

		log.Fatalf(api.ValidateEnvVar(validateErr))
	}

	if err := validator.New().Struct(sessions_db); err != nil {
		var validateErr validator.ValidationErrors
		if !errors.As(err, &validateErr) {
			log.Fatalf("cannot to validate sessions_db init data")
		}

		log.Fatalf(api.ValidateEnvVar(validateErr))
	}

	cfg.MainStorageConnString = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		main_db.DB_USERNAME,
		main_db.DB_PASSWORD,
		main_db.DB_HOST,
		"5432",
		main_db.DB_NAME,
	)

	cfg.SessionsStorageConnString = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		sessions_db.DB_USERNAME,
		sessions_db.DB_PASSWORD,
		sessions_db.DB_HOST,
		"5432",
		sessions_db.DB_NAME,
	)

	secretKey := os.Getenv("SECRET_KEY")
	if secretKey == "" {
		log.Fatalf("env SECRET_KEY is required")
	}

	cfg.Secret = secretKey

	return &cfg
}

func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
