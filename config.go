package main

import (
	"log"
	"os"
	"regexp"

	"github.com/caarlos0/env"
)

type Config struct {
	DATA_DIR              string   `env:"DATA_DIR,required"`
	DEFAULT_IMAGE_QUALITY int      `env:"IMAGE_QUALITY" envDefault:"95"`
	SERVER_PORT           int      `env:"SERVER_PORT" envDefault:"8080"`
	TOKEN                 string   `env:"TOKEN" envDefault=""`
	DEBUG                 bool     `env:"DEBUG"`
	VALID_SIZES           []string `env:"VALID_SIZES" envSeparator:":"`
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
	sizePattern := regexp.MustCompile("([0-9]{2,4})x([0-9]{2,4})")
	for _, size := range cfg.VALID_SIZES {
		match := sizePattern.FindAllString(size, -1)
		if len(match) != 1 {
			log.Fatalf("Image size %f is not valid. Try use WIDTHxHEIGHT format.", size)
		}
	}

	return &cfg
}
