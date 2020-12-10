package main

import (
	"fmt"
	"log"

	"mime/multipart"
	"net/http"
)

func ValidateImage(header *multipart.FileHeader) bool {
	file, err := header.Open()
	defer file.Close()
	if err != nil {
		log.Println(err)
		return false
	}
	buff := make([]byte, 512)
	if _, err = file.Read(buff); err != nil {
		log.Println(err)
		return false
	}
	fileType := http.DetectContentType(buff)

	switch fileType {
	case "image/jpeg", "image/jpg", "image/gif", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

func ValidateImageSize(width int, height int, config *Config) bool {
	s := fmt.Sprintf("%dx%d", width, height)
	for _, size := range config.VALID_SIZES {
		if size == s {
			return true
		}
	}
	return false
}
