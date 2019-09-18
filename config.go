package main

import (
	"log"

	"github.com/caarlos0/env"
)

type Config struct {
	IMAGES_ROOT           string `env:"IMAGES_ROOT,required"`
	DEFAULT_IMAGE_QUALITY int    `env:"IMAGE_QUALITY" envDefault:"95"`
	SERVER_PORT           int    `env:"SERVER_PORT" envDefault:"8080"`
}

func GetConfig() *Config {
	cfg := Config{}

	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("%+v\n", err)
	}
	return &cfg
}
