package main

import (
	"os"
	"strings"
	"testing"

	"github.com/matryer/is"
)

func TestParseConfig(t *testing.T) {
	is := is.New(t)
	config_file := strings.NewReader(`
data_directory:
  /tmp/webp-server/
default_image_quality:
  80
server_address:
  127.0.0.1:9000
token:
  abcdefg
valid_image_sizes:
  - 200x200
  - 500x500
  - 600x600
valid_image_qualities:
  - 90
  - 95
  - 100
max_uploaded_image_size:
  3
http_cache_ttl:
  10
debug:
  true
convert_concurrency:
  3
`)
	defer os.RemoveAll("/tmp/webp-server")
	cfg := ParseConfig(config_file)
	expected := &Config{
		DataDir:              "/tmp/webp-server/",
		DefaultImageQuality:  80,
		ServerAddress:        "127.0.0.1:9000",
		Token:                "abcdefg",
		ValidImageSizes:      []string{"200x200", "500x500", "600x600"},
		ValidImageQualities:  []int{90, 95, 100},
		MaxUploadedImageSize: 3,
		HttpCacheTTL:         10,
		Debug:                true,
		ConvertConcurrency:   3,
	}

	is.Equal(cfg, expected)
	if tok := os.Getenv("WEBP_SERVER_TOKEN"); len(tok) == 0 {
		os.Setenv("WEBP_SERVER_TOKEN", "123")
		defer os.Unsetenv("WEBP_SERVER_TOKEN")
	}
	_, err := config_file.Seek(0, 0)
	is.NoErr(err)
	cfg = ParseConfig(config_file)
	is.Equal(cfg.Token, os.Getenv("WEBP_SERVER_TOKEN"))
}
