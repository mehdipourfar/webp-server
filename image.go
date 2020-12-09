package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"github.com/valyala/fasthttp"
	bimg "gopkg.in/h2non/bimg.v1"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func init() {
	bimg.VipsCacheSetMax(0)
	bimg.VipsCacheSetMaxMem(0)
}

var imageUrlRegex = regexp.MustCompile("/image/(?P<filterParams>[0-9a-z,=-]+)/(?P<imageId>[0-9a-zA-Z_-]+)")

const (
	FIT_COVER      = "cover"
	FIT_CONTAIN    = "contain"
	FIT_SCALE_DOWN = "scale-down"

	FORMAT_AUTO     = "auto"
	FORMAT_ORIGINAL = "original"
	FORMAT_JPEG     = "jpeg"
	FORMAT_PNG      = "png"
	FORMAT_WEBP     = "webp"
)

type ImageParams struct {
	ImageId      string
	Width        int
	Height       int
	Format       string
	Fit          string
	Quality      int
	WebpAccepted bool
}

func GetImageParamsFromRequest(header *fasthttp.RequestHeader, config *Config) (*ImageParams, error) {
	match := imageUrlRegex.FindSubmatch(header.RequestURI())
	if len(match) != 3 {
		return nil, fmt.Errorf("Not Match")
	}

	imageId := string(match[2])

	params := &ImageParams{
		ImageId:      imageId,
		Fit:          FIT_CONTAIN,
		Format:       FORMAT_AUTO,
		Quality:      config.DEFAULT_IMAGE_QUALITY,
		WebpAccepted: bytes.Contains(header.Peek("accept"), []byte("webp")),
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
			if fit := keyVal[1]; fit == FIT_COVER || fit == FIT_CONTAIN || fit == FIT_SCALE_DOWN {
				params.Fit = fit
			} else {
				return nil, fmt.Errorf("Supported fits are cover, contain and scale-down")
			}
		case "format", "f":
			f := keyVal[1]
			if f == FORMAT_WEBP || f == FORMAT_JPEG || f == FORMAT_PNG || f == FORMAT_AUTO || f == FORMAT_ORIGINAL {
				params.Format = f
			} else {
				return nil, fmt.Errorf("Supported formats are auto, original, webp, jpeg and png")
			}
		default:
			return nil, fmt.Errorf("Invalid filter key: %s", keyVal[0])
		}
	}

	return params, nil
}

func ImageIdToFilePath(dataDir string, imageId string) (parentDir string, filePath string) {
	parentDir = fmt.Sprintf("images/%s/%s", imageId[1:2], imageId[3:5])
	parentDir = filepath.Join(dataDir, parentDir)
	filePath = fmt.Sprintf("%s/%s", parentDir, imageId)
	return
}

func (i *ImageParams) GetMd5() string {
	key := fmt.Sprintf(
		"%s:%d:%d:%s:%s:%d:%t",
		i.ImageId,
		i.Width,
		i.Height,
		i.Format,
		i.Fit,
		i.Quality,
		i.WebpAccepted,
	)
	h := md5.New()
	io.WriteString(h, key)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (i *ImageParams) GetCachePath(dataDir string) (parentDir string, filePath string) {
	fileName := i.GetMd5()

	parentDir = fmt.Sprintf("caches/%s/%s", fileName[31:32], fileName[29:31])
	parentDir = filepath.Join(dataDir, parentDir)
	filePath = fmt.Sprintf("%s/%s", parentDir, fileName)
	return
}

func (params *ImageParams) ToBimgOptions(size *bimg.ImageSize, imageType bimg.ImageType) *bimg.Options {
	options := &bimg.Options{
		Quality: params.Quality,
	}

	if params.Fit == FIT_COVER {
		options.Crop = true
		options.Embed = true
		options.Width = params.Width
		options.Height = params.Height
	}
	if params.Fit == FIT_CONTAIN || params.Fit == FIT_SCALE_DOWN {
		if params.Width == 0 || params.Height == 0 {
			options.Width = params.Width
			options.Height = params.Height
		} else {
			imageRatio := float32(size.Width) / float32(size.Height)
			wantedRatio := float32(params.Width) / float32(params.Height)

			if wantedRatio < imageRatio {
				options.Width = params.Width
			} else {
				options.Height = params.Height
			}
		}

		if params.Fit == FIT_SCALE_DOWN {
			if options.Width > size.Width {
				options.Width = size.Width
			}
			if options.Height > size.Height {
				options.Height = size.Height
			}
		}
	}

	switch params.Format {
	case FORMAT_AUTO:
		if params.WebpAccepted {
			options.Type = bimg.WEBP
		} else {
			options.Type = bimg.JPEG
		}
	case FORMAT_ORIGINAL:
		options.Type = imageType
	case FORMAT_JPEG:
		options.Type = bimg.JPEG
	case FORMAT_WEBP:
		options.Type = bimg.WEBP
	case FORMAT_PNG:
		options.Type = bimg.PNG
	}
	return options
}

func Convert(fileBuffer []byte, params *ImageParams) ([]byte, bimg.ImageType, error) {
	imageType := bimg.DetermineImageType(fileBuffer)

	if imageType == bimg.GIF {
		// ignore gif conversion
		return fileBuffer, bimg.GIF, nil
	}
	img := bimg.NewImage(fileBuffer)
	size, err := img.Size()
	if err != nil {
		return nil, imageType, err
	}

	options := params.ToBimgOptions(&size, imageType)
	newImage, err := img.Process(*options)
	if err != nil {
		return nil, options.Type, err
	}
	return newImage, options.Type, nil
}
