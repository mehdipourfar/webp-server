package main

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {
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
`)
	cfg := ParseConfig(config_file)
	expected := &Config{
		DataDir:              "/tmp/webp-server/",
		DefaultImageQuality:  80,
		ServerAddress:        "127.0.0.1:9000",
		Token:                "abcdefg",
		ValidImageSizes:      []string{"200x200", "500x500", "600x600"},
		ValidImageQualities:  []int{90, 95, 100},
		MaxUploadedImageSize: 3,
	}

	if !reflect.DeepEqual(cfg, expected) {
		t.Errorf("Expected config to be\n%+v\nbut result is\n%+v\n",
			expected, cfg)
	}
	os.RemoveAll("/tmp/webp-server")
}
