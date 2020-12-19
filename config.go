package main

import (
	"log"
	"net"
	"os"
	"regexp"

	"github.com/caarlos0/env"
)

type Config struct {
	DATA_DIR              string   `env:"DATA_DIR,required"`
	DEFAULT_IMAGE_QUALITY int      `env:"IMAGE_QUALITY" envDefault:"95"`
	SERVER_PORT           int      `env:"SERVER_PORT" envDefault:"8080"`
	SERVER_ADDR           string   `env:"SERVER_ADDR" envDefault:"127.0.0.1"`
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
			log.Fatalf("Image size %s is not valid. Try use WIDTHxHEIGHT format.", size)
		}
	}
	if net.ParseIP(cfg.SERVER_ADDR) == nil {
		log.Fatalf("Address %s is not a valid IP.", cfg.SERVER_ADDR)
	}

	return &cfg
}
