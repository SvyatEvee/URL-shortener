package config

import (
	"flag"
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"os"
	"time"
)

type Config struct {
	Env                 string        `yaml:"env" env-default:"local"`
	MainStoragePath     string        `yaml:"mainStorage_path" env-required:"true"`
	SessionsStoragePath string        `yaml:"sessions_storage_path" env-required:"true"`
	Secret              string        `yaml:"secret"`
	AccessTokenTTL      time.Duration `yaml:"access_token_ttl" env-required:"true"`
	RefreshTokenTTL     time.Duration `yaml:"refresh_token_ttl" env-required:"true"`
	GRPC                GRPCConfig    `yaml:"grpc"`
}

type GRPCConfig struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
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
