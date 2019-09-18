package main

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var re *regexp.Regexp

func init() {
	re = regexp.MustCompile("/image/(?P<FilterParams>[0-9a-z,=-]+)/(?P<FilePath>.+)")
}

const (
	FIT_COVER      = "cover"
	FIT_CONTAIN    = "contain"
	FIT_SCALE_DOWN = "scale-down"

	FORMAT_AUTO = "auto"
	FORMAT_JPEG = "jpeg"
	FORMAT_PNG  = "png"
	FORMAT_WEBP = "webp"
)

type ImageParams struct {
	FilePath     string
	Width        int
	Height       int
	Format       string
	Fit          string
	Quality      int
	WebpAccepted bool
}

func GetParamsFromUri(reqUri []byte) (*ImageParams, error) {
	match := re.FindSubmatch(reqUri)
	if len(match) != 3 {
		return nil, errors.New("Not Match")
	}
	params := &ImageParams{
		FilePath: string(match[2]),
		Fit:      FIT_CONTAIN,
		Format:   FORMAT_AUTO,
		Quality:  config.DEFAULT_IMAGE_QUALITY,
	}

	var err error

	for _, item := range strings.Split(string(match[1]), ",") {
		keyVal := strings.Split(item, "=")
		if len(keyVal) != 2 {
			return nil, fmt.Errorf("Bad Filter Param, %v", keyVal)
		}
		switch keyVal[0] {
		case "width", "w":
			if params.Width, err = strconv.Atoi(keyVal[1]); err != nil {
				return nil, fmt.Errorf("Width should be integer")
			}
		case "height", "h":
			if params.Height, err = strconv.Atoi(keyVal[1]); err != nil {
				return nil, fmt.Errorf("Height should be integer")
			}
		case "quality", "q":
			if params.Quality, err = strconv.Atoi(keyVal[1]); err != nil {
				return nil, fmt.Errorf("Quality should be integer")
			}
		case "fit":
			if fit := keyVal[1]; fit == FIT_COVER || fit == FIT_CONTAIN {
				params.Fit = fit
			} else {
				return nil, fmt.Errorf("Supported fits are cover and contain")
			}
		case "format", "f":
			if format := keyVal[1]; format == FORMAT_WEBP || format == FORMAT_JPEG || format == FORMAT_PNG {
				params.Format = format
			} else {
				return nil, fmt.Errorf("Supported fits are cover and contain")
			}
		default:
			return nil, fmt.Errorf("Invalid filter key: %s", keyVal[0])
		}
	}
	return params, nil
}
