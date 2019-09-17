package main

import (
	bimg "gopkg.in/h2non/bimg.v1"

	"bytes"
	"path/filepath"
)

func Convert(params *ImageParams) ([]byte, error) {
	input := filepath.Join(config.IMAGES_ROOT, params.FileName)
	buffer, err := bimg.Read(input)
	if err != nil {
		return nil, err
	}
	if bytes.HasPrefix(buffer, []byte("GIF")) {
		return buffer, nil
	}
	options := bimg.Options{
		Quality: 95,
		Width:   params.Width,
		Height:  params.Height,
		Crop:    params.Crop,
	}
	if !params.Crop {
		options.Embed = true
	}

	if params.Webp {
		options.Type = bimg.WEBP
	} else {
		options.Type = bimg.JPEG
	}

	newImage, err := bimg.NewImage(buffer).Process(options)
	if err != nil {
		return nil, err
	}
	return newImage, nil
}
