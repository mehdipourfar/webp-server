package main

import (
	bimg "gopkg.in/h2non/bimg.v1"

	"path/filepath"
)

func Convert(params *ImageParams) ([]byte, error) {
	input := filepath.Join(config.IMAGES_ROOT, params.FileName)
	buffer, err := bimg.Read(input)
	if err != nil {
		return nil, err
	}
	imageType := bimg.DetermineImageType(buffer)

	if imageType == bimg.GIF {
		return buffer, nil
	}
	options := bimg.Options{
		Quality: config.IMAGE_QUALITY,
		Crop:    params.Crop,
	}
	img := bimg.NewImage(buffer)
	if !params.Crop { // contain
		size, err := img.Size()
		if err != nil {
			return nil, err
		}
		if size.Width > size.Height {
			options.Width = params.Width
		} else {
			options.Height = params.Height
		}
	} else { // cover
		options.Width = params.Width
		options.Height = params.Height
		options.Embed = true
	}

	if params.Webp {
		options.Type = bimg.WEBP
	} else {
		options.Type = imageType
	}

	newImage, err := img.Process(options)
	if err != nil {
		return nil, err
	}
	return newImage, nil
}
