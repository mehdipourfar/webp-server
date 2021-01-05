package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	"gopkg.in/yaml.v2"
)

//Config is global configuration of the server
type Config struct {
	DataDir              string   `yaml:"data_directory"`
	DefaultImageQuality  int      `yaml:"default_image_quality"`
	ServerAddress        string   `yaml:"server_address"`
	Token                string   `yaml:"token"`
	ValidImageSizes      []string `yaml:"valid_image_sizes"`
	ValidImageQualities  []int    `yaml:"valid_image_qualities"`
	MaxUploadedImageSize int      `yaml:"max_uploaded_image_size"` // in megabytes
	HTTPCacheTTL         int      `yaml:"http_cache_ttl"`
	LogPath              string   `yaml:"log_path"`
	Debug                bool     `yaml:"debug"`
	ConvertConcurrency   int      `yaml:"convert_concurrency"`
}

func getDefaultConfig() *Config {
	return &Config{
		DefaultImageQuality:  95,
		ServerAddress:        "127.0.0.1:8080",
		ValidImageSizes:      []string{"300x300", "500x500"},
		MaxUploadedImageSize: 4,
		HTTPCacheTTL:         2592000,
		ConvertConcurrency:   runtime.NumCPU(),
	}

}

func parseConfig(file io.Reader) (*Config, error) {
	cfg := getDefaultConfig()
	buf, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("%+v\n", err)
	}
	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		return nil, fmt.Errorf("Invalid Config File: %v", err)
	}

	if token := os.Getenv("WEBP_SERVER_TOKEN"); len(token) != 0 {
		cfg.Token = token
	}

	if cfg.DataDir == "" {
		return nil, fmt.Errorf("Set data_directory in your config file.")
	}

	if !filepath.IsAbs(cfg.DataDir) {
		return nil, fmt.Errorf("Absolute path for data_dir needed but got: %s", cfg.DataDir)
	}

	if len(cfg.LogPath) > 0 && !filepath.IsAbs(cfg.LogPath) {
		return nil, fmt.Errorf("Absolute path for log_path needed but got: %s", cfg.LogPath)
	}

	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("%+v\n", err)
	}

	sizePattern := regexp.MustCompile("([0-9]{1,4})x([0-9]{1,4})")
	for _, size := range cfg.ValidImageSizes {
		match := sizePattern.FindAllString(size, -1)
		if len(match) != 1 {
			return nil, fmt.Errorf("Image size %s is not valid. Try use WIDTHxHEIGHT format.", size)
		}
	}

	if cfg.DefaultImageQuality < 10 || cfg.DefaultImageQuality > 100 {
		return nil, fmt.Errorf("Default image quality should be 10 < q < 100.")
	}

	if cfg.ConvertConcurrency <= 0 {
		return nil, fmt.Errorf("Convert Concurrency should be greater than zero")
	}
	return cfg, nil
}
