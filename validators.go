package main

import (
	"fmt"
	"log"

	"mime/multipart"
	"net/http"
)

func ValidateImage(header *multipart.FileHeader) bool {
	file, err := header.Open()
	if err != nil {
		log.Println(err)
		return false
	}
	defer file.Close()
	buff := make([]byte, 512)
	if _, err = file.Read(buff); err != nil {
		log.Println(err)
		return false
	}
	ct := http.DetectContentType(buff)

	switch ct {
	case "image/jpeg", "image/jpg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

func ValidateImageParams(imageParams *ImageParams, config *Config) error {
	valid_size := false
	valid_quality := imageParams.Quality == 0

	image_size := fmt.Sprintf("%dx%d", imageParams.Width, imageParams.Height)
	for _, size := range config.ValidImageSizes {
		if size == image_size {
			valid_size = true
			break
		}
	}

	if imageParams.Quality <= 100 && imageParams.Quality >= 10 {
		for _, val := range config.ValidImageQualities {
			if val == imageParams.Quality {
				valid_quality = true
				break
			}
		}
	}

	if !valid_size {
		return fmt.Errorf(
			"size=%dx%d is not supported by server. Contact server admin.",
			imageParams.Width, imageParams.Height)
	}

	if !valid_quality {
		return fmt.Errorf(
			"quality=%d is not supported by server. Contact server admin.",
			imageParams.Quality)
	}
	return nil
}
