package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/matryer/is"
)

func TestparseConfig(t *testing.T) {
	is := is.New(t)
	configFile := strings.NewReader(`
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
	cfg, err := parseConfig(configFile)
	if err != nil {
		t.Fatal(err)
	}
	expected := &Config{
		DataDir:              "/tmp/webp-server/",
		DefaultImageQuality:  80,
		ServerAddress:        "127.0.0.1:9000",
		Token:                "abcdefg",
		ValidImageSizes:      []string{"200x200", "500x500", "600x600"},
		ValidImageQualities:  []int{90, 95, 100},
		MaxUploadedImageSize: 3,
		HTTPCacheTTL:         10,
		Debug:                true,
		ConvertConcurrency:   3,
	}

	is.Equal(cfg, expected)
	if tok := os.Getenv("WEBP_SERVER_TOKEN"); len(tok) == 0 {
		os.Setenv("WEBP_SERVER_TOKEN", "123")
		defer os.Unsetenv("WEBP_SERVER_TOKEN")
	}
	_, err = configFile.Seek(0, 0)
	is.NoErr(err)
	cfg, err = parseConfig(configFile)
	if err != nil {
		t.Fatal(err)
	}
	is.Equal(cfg.Token, os.Getenv("WEBP_SERVER_TOKEN"))
}

func TestParseConfigErrors(t *testing.T) {
	tt := []struct {
		name string
		file io.Reader
		err  error
	}{
		{
			name: "parse_error",
			file: strings.NewReader("----"),
			err:  fmt.Errorf("Invalid Config File: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `----` into main.Config"),
		},
		{
			name: "empty_data_dir",
			file: strings.NewReader("server_address: 127.0.0.1:8080"),
			err:  fmt.Errorf("Set data_directory in your config file."),
		},
		{
			name: "non_absolute_data_dir",
			file: strings.NewReader("data_directory: ~/data"),
			err:  fmt.Errorf("Absolute path for data_dir needed but got: ~/data"),
		},
		{
			name: "non_absolute_log_path",
			file: strings.NewReader("data_directory: /tmp/\nlog_path: ~/log"),
			err:  fmt.Errorf("Absolute path for log_path needed but got: ~/log"),
		},
		{
			name: "invalid_image_size",
			file: strings.NewReader("data_directory: /tmp/\nvalid_image_sizes:\n  - 300x"),
			err:  fmt.Errorf("Image size 300x is not valid. Try use WIDTHxHEIGHT format."),
		},
		{
			name: "invalid_default_quality",
			file: strings.NewReader("data_directory: /tmp/\ndefault_image_quality: 120"),
			err:  fmt.Errorf("Default image quality should be 10 < q < 100."),
		},
		{
			name: "invalid_convert_concurrency",
			file: strings.NewReader("data_directory: /tmp/\nconvert_concurrency: 0"),
			err:  fmt.Errorf("Convert Concurrency should be greater than zero"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			is := is.NewRelaxed(t)
			_, err := parseConfig(tc.file)
			is.True(err != nil)
			is.Equal(err, tc.err)
		})
	}

}
