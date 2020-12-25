package main

import (
	"crypto/md5"
	"fmt"
	bimg "gopkg.in/h2non/bimg.v1"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func init() {
	bimg.VipsCacheSetMax(0)
	bimg.VipsCacheSetMaxMem(0)
}

const (
	FIT_COVER      = "cover"
	FIT_CONTAIN    = "contain"
	FIT_SCALE_DOWN = "scale-down"
)

type ImageParams struct {
	ImageId      string
	Width        int
	Height       int
	Fit          string
	Quality      int
	WebpAccepted bool
}

func CreateImageParams(imageId, options string, webpAccepted bool, config *Config) (*ImageParams, error) {
	params := &ImageParams{
		ImageId:      imageId,
		Fit:          FIT_CONTAIN,
		Quality:      config.DefaultImageQuality,
		WebpAccepted: webpAccepted,
	}

	var err error

	for _, op := range strings.Split(options, ",") {
		kv := strings.Split(op, "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("Invalid param: %s", op)
		}
		key, val := kv[0], kv[1]

		switch key {
		case "width", "w":
			if params.Width, err = strconv.Atoi(val); err != nil {
				return nil, fmt.Errorf("Width should be integer")
			}
		case "height", "h":
			if params.Height, err = strconv.Atoi(val); err != nil {
				return nil, fmt.Errorf("Height should be integer")
			}
		case "fit":
			switch val {
			case FIT_CONTAIN, FIT_COVER, FIT_SCALE_DOWN:
				params.Fit = val
			default:
				return nil, fmt.Errorf("Supported fits are cover, contain and scale-down")
			}
		case "quality", "q":
			params.Quality, err = strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("Quality should be integer")
			}
		default:
			return nil, fmt.Errorf("Invalid filter key: %s", key)
		}
	}

	return params, nil
}

func ImageIdToFilePath(dataDir string, imageId string) string {
	parentDir := fmt.Sprintf("images/%s/%s", imageId[1:2], imageId[3:5])
	parentDir = filepath.Join(dataDir, parentDir)
	return fmt.Sprintf("%s/%s", parentDir, imageId)
}

func (i *ImageParams) GetMd5() string {
	key := fmt.Sprintf(
		"%s:%d:%d:%s:%d:%t",
		i.ImageId,
		i.Width,
		i.Height,
		i.Fit,
		i.Quality,
		i.WebpAccepted,
	)
	h := md5.New()
	io.WriteString(h, key)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (i *ImageParams) GetCachePath(dataDir string) string {
	md5Sum := i.GetMd5()
	fileName := fmt.Sprintf("%s-%s", i.ImageId, md5Sum)
	parentDir := fmt.Sprintf("caches/%s/%s", md5Sum[31:32], md5Sum[29:31])
	parentDir = filepath.Join(dataDir, parentDir)
	return fmt.Sprintf("%s/%s", parentDir, fileName)
}

func (params *ImageParams) ToBimgOptions(size *bimg.ImageSize) *bimg.Options {
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
			requestRatio := float32(params.Width) / float32(params.Height)

			if requestRatio < imageRatio {
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

	if params.WebpAccepted {
		options.Type = bimg.WEBP
	} else {
		options.Type = bimg.JPEG
	}
	return options
}

func Convert(inputPath, outputPath string, params *ImageParams) error {
	imgBuffer, err := ioutil.ReadFile(inputPath)
	if err != nil {
		return err
	}
	img := bimg.NewImage(imgBuffer)
	size, err := img.Size()
	if err != nil {
		return err
	}

	options := params.ToBimgOptions(&size)
	newImage, err := img.Process(*options)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(outputPath, newImage, 0604); err != nil {
		return err
	}
	return nil
}
