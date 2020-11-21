package main

import (
	"log"
	"os"

	"github.com/caarlos0/env"
)

type Config struct {
	DATA_DIR              string `env:"DATA_DIR,required"`
	DEFAULT_IMAGE_QUALITY int    `env:"IMAGE_QUALITY" envDefault:"95"`
	SERVER_PORT           int    `env:"SERVER_PORT" envDefault:"8080"`
	TOKEN                 string `env:"TOKEN" envDefault=""`
}

func GetConfig() *Config {
	var err error
	cfg := Config{}

	if err = env.Parse(&cfg); err != nil {
		log.Fatalf("%+v\n", err)
	}

	if err = os.MkdirAll(cfg.DATA_DIR, 0755); err != nil {
		log.Fatalf("%+v\n", err)
	}

	return &cfg
}
