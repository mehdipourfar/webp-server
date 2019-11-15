package main

import (
	bimg "gopkg.in/h2non/bimg.v1"

	"path/filepath"
)

func init() {
	bimg.VipsCacheSetMax(0)
	bimg.VipsCacheSetMaxMem(0)
}

func Convert(params *ImageParams) ([]byte, bimg.ImageType, error) {
	input := filepath.Join(config.IMAGES_ROOT, params.FilePath)
	buffer, err := bimg.Read(input)
	if err != nil {
		return nil, bimg.UNKNOWN, err
	}
	imageType := bimg.DetermineImageType(buffer)

	if imageType == bimg.GIF {
		// ignore gif conversion
		return buffer, bimg.GIF, nil
	}

	options := bimg.Options{
		Quality: params.Quality,
	}

	if params.Fit == FIT_COVER {
		options.Crop = true
	}
	img := bimg.NewImage(buffer)
	if params.Fit == FIT_CONTAIN {
		size, err := img.Size()
		if err != nil {
			return nil, imageType, err
		}
		if size.Width > size.Height {
			options.Width = params.Width
		} else {
			options.Height = params.Height
		}
	} else {
		options.Width = params.Width
		options.Height = params.Height
		options.Embed = true
	}

	switch params.Format {
	case FORMAT_AUTO:
		if params.WebpAccepted {
			options.Type = bimg.WEBP
		} else {
			options.Type = imageType
		}
	case FORMAT_JPEG:
		options.Type = bimg.JPEG
	case FORMAT_WEBP:
		options.Type = bimg.WEBP
	case FORMAT_PNG:
		options.Type = bimg.PNG
	}

	newImage, err := img.Process(options)
	if err != nil {
		return nil, options.Type, err
	}
	return newImage, options.Type, nil
}
