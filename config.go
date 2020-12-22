package main

import (
	"io/ioutil"
	"log"
	"os"
	"regexp"

	"gopkg.in/yaml.v2"
)

type Config struct {
	DataDir             string   `yaml:"data_directory"`
	DefaultImageQuality int      `yaml:"default_image_quality"`
	ServerAddress       string   `yaml:"server_address"`
	Token               string   `yaml:"token"`
	ValidImageSizes     []string `yaml:"valid_image_sizes"`
	MaxRequestBodySize  int      `yaml:"max_request_body_size"` // in megabytes
}

func ParseConfig(path string) *Config {
	cfg := Config{
		DefaultImageQuality: 95,
		ServerAddress:       "127.0.0.1:8080",
		ValidImageSizes:     []string{"300x300", "500x500"},
		MaxRequestBodySize:  4,
	}

	buf, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("%+v\n", err)
	}
	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		log.Fatalf("Invalid Config File: %v\n", err)
	}

	if cfg.DataDir == "" {
		log.Fatalf("Set data_directory in your config file.")
	}

	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		log.Fatalf("%+v\n", err)
	}
	sizePattern := regexp.MustCompile("([0-9]{1,4})x([0-9]{1,4})")
	for _, size := range cfg.ValidImageSizes {
		match := sizePattern.FindAllString(size, -1)
		if len(match) != 1 {
			log.Fatalf("Image size %s is not valid. Try use WIDTHxHEIGHT format.", size)
		}
	}
	return &cfg
}
