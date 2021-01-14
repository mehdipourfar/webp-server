package main

import (
	"crypto/md5"
	"fmt"
	"github.com/valyala/bytebufferpool"
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
	//FitCover is used for resize and crop
	FitCover = "cover"
	//FitContain is used for resize and keep aspect ratio
	FitContain = "contain"
	//FitScaleDown is like FitContain except that it prevent image to be enlarged
	FitScaleDown = "scale-down"
)

//ImageParams is request properties for image conversion
type ImageParams struct {
	ImageID      string
	Width        int
	Height       int
	Fit          string
	Quality      int
	WebpAccepted bool
}

func createImageParams(imageID, options string, webpAccepted bool, config *Config) (*ImageParams, error) {
	params := &ImageParams{
		ImageID:      imageID,
		Fit:          FitContain,
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
			case FitContain, FitCover, FitScaleDown:
				params.Fit = val
			default:
				return nil, fmt.Errorf("Supported fits are cover, contain and scale-down")
			}
		case "quality", "q":
			if params.Quality, err = strconv.Atoi(val); err != nil {
				return nil, fmt.Errorf("Quality should be integer")
			}
		default:
			return nil, fmt.Errorf("Invalid filter key: %s", key)
		}
	}

	return params, nil
}

func getFilePathFromImageID(dataDir string, imageID string) string {
	parentDir := fmt.Sprintf("images/%s/%s", imageID[1:2], imageID[3:5])
	parentDir = filepath.Join(dataDir, parentDir)
	return fmt.Sprintf("%s/%s", parentDir, imageID)
}

func (params *ImageParams) getMd5() string {
	key := fmt.Sprintf(
		"%s:%d:%d:%s:%d:%t",
		params.ImageID,
		params.Width,
		params.Height,
		params.Fit,
		params.Quality,
		params.WebpAccepted,
	)
	h := md5.New()
	_, err := io.WriteString(h, key)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (params *ImageParams) getCachePath(dataDir string) string {
	md5Sum := params.getMd5()
	fileName := fmt.Sprintf("%s-%s", params.ImageID, md5Sum)
	parentDir := fmt.Sprintf("caches/%s/%s", md5Sum[31:32], md5Sum[29:31])
	parentDir = filepath.Join(dataDir, parentDir)
	return fmt.Sprintf("%s/%s", parentDir, fileName)
}

func (params *ImageParams) toBimgOptions(size *bimg.ImageSize) *bimg.Options {
	options := &bimg.Options{
		Quality: params.Quality,
	}

	if params.Fit == FitCover {
		options.Crop = true
		options.Embed = true
		options.Width = params.Width
		options.Height = params.Height
	}
	if params.Fit == FitContain || params.Fit == FitScaleDown {
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

		if params.Fit == FitScaleDown {
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

func convert(inputPath, outputPath string, params *ImageParams) error {
	f, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	buffer := bytebufferpool.Get()
	defer bytebufferpool.Put(buffer)
	_, err = buffer.ReadFrom(f)
	f.Close()
	if err != nil {
		return err
	}

	img := bimg.NewImage(buffer.B)
	size, err := img.Size()
	if err != nil {
		return err
	}

	options := params.toBimgOptions(&size)
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
