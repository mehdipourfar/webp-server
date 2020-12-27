package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v2"
)

type Config struct {
	DataDir              string   `yaml:"data_directory"`
	DefaultImageQuality  int      `yaml:"default_image_quality"`
	ServerAddress        string   `yaml:"server_address"`
	Token                string   `yaml:"token"`
	ValidImageSizes      []string `yaml:"valid_image_sizes"`
	ValidImageQualities  []int    `yaml:"valid_image_qualities"`
	MaxUploadedImageSize int      `yaml:"max_uploaded_image_size"` // in megabytes
	HttpCacheTTL         int      `yaml:"http_cache_ttl"`
	LogPath              string   `yaml:"log_path"`
	Debug                bool     `yaml:"debug"`
}

func ParseConfig(file io.Reader) *Config {
	cfg := Config{
		DefaultImageQuality:  95,
		ServerAddress:        "127.0.0.1:8080",
		ValidImageSizes:      []string{"300x300", "500x500"},
		MaxUploadedImageSize: 4,
		HttpCacheTTL:         2592000,
	}

	buf, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("%+v\n", err)
	}
	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		log.Fatalf("Invalid Config File: %v\n", err)
	}

	if cfg.DataDir == "" {
		log.Fatalf("Set data_directory in your config file.")
	}

	if !filepath.IsAbs(cfg.DataDir) {
		log.Fatalf("Absolute path for data_dir is needed but got: %s", cfg.DataDir)
	}

	if len(cfg.LogPath) > 0 && !filepath.IsAbs(cfg.LogPath) {
		log.Fatalf("Absolute path for log_path is needed but got: %s", cfg.DataDir)
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

	if cfg.DefaultImageQuality < 10 || cfg.DefaultImageQuality > 100 {
		log.Fatal("Default image quality should be 10 < q < 100.")
	}
	return &cfg
}
