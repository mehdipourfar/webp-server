package main

import (
	"fmt"
	"github.com/matryer/is"
	bimg "gopkg.in/h2non/bimg.v1"
	"testing"
)

func (p *ImageParams) String() string {
	return fmt.Sprintf(
		"id:%s,width:%d,height:%d,fit:%s,quality:%d,webp_accepted:%t",
		p.ImageID, p.Width, p.Height, p.Fit, p.Quality, p.WebpAccepted,
	)
}

func bimgOptsToString(o *bimg.Options) string {
	return fmt.Sprintf(
		"type:%d,width:%d,height:%d,crop:%t,embed:%t",
		o.Type, o.Width, o.Height, o.Crop, o.Embed,
	)
}

func bimgOptsAreEqual(o1 *bimg.Options, o2 *bimg.Options) bool {
	return o1.Type == o2.Type && o1.Width == o2.Width &&
		o1.Height == o2.Height && o1.Crop == o2.Crop && o1.Embed == o2.Embed
}

func TestImagePath(t *testing.T) {
	is := is.New(t)
	imagePath := getFilePathFromImageID("/tmp/media", "FyBmW7C2f")
	is.Equal(imagePath, "/tmp/media/images/y/mW/FyBmW7C2f")
}

func TestCachePath(t *testing.T) {
	is := is.New(t)
	params := &ImageParams{
		ImageID:      "NG4uQBa2f",
		Width:        100,
		Height:       100,
		Fit:          "cover",
		Quality:      90,
		WebpAccepted: true,
	}

	is.Equal(params.getMd5(), "c64dda22268336d2c246899c2bc79005")
	is.Equal(
		params.getCachePath("/tmp/media/"),
		"/tmp/media/caches/5/00/NG4uQBa2f-c64dda22268336d2c246899c2bc79005",
	)
}

func TestGetParamsFromUri(t *testing.T) {
	config := &Config{
		DataDir:             "/tmp/media/",
		DefaultImageQuality: 50,
		ValidImageQualities: []int{50, 90, 95},
	}

	tt := []struct {
		testID         int
		imageID        string
		options        string
		webpAccepted   bool
		expectedParams *ImageParams
		err            error
	}{
		{
			testID:       1,
			imageID:      "NG4uQBa2f",
			options:      "w=500,h=500,fit=contain",
			webpAccepted: false,
			expectedParams: &ImageParams{
				ImageID:      "NG4uQBa2f",
				Fit:          "contain",
				Width:        500,
				Height:       500,
				Quality:      50,
				WebpAccepted: false,
			},
			err: nil,
		},
		{
			testID:       2,
			imageID:      "NG4uQBa2f",
			options:      "w=300,h=300,fit=contain",
			webpAccepted: false,
			expectedParams: &ImageParams{
				ImageID:      "NG4uQBa2f",
				Fit:          "contain",
				Width:        300,
				Height:       300,
				Quality:      50,
				WebpAccepted: false,
			},
			err: nil,
		},
		{
			testID:       3,
			imageID:      "NG4uQBa2f",
			options:      "w=300,h=300,fit=contain",
			webpAccepted: true,
			expectedParams: &ImageParams{
				ImageID:      "NG4uQBa2f",
				Fit:          "contain",
				Width:        300,
				Height:       300,
				Quality:      50,
				WebpAccepted: true,
			},
			err: nil,
		},
		{
			testID:       4,
			imageID:      "NG4uQBa2f",
			options:      "w=300,h=300,fit=cover",
			webpAccepted: true,
			expectedParams: &ImageParams{
				ImageID:      "NG4uQBa2f",
				Fit:          "cover",
				Width:        300,
				Height:       300,
				Quality:      50,
				WebpAccepted: true,
			},
			err: nil,
		},
		{
			testID:       7,
			imageID:      "NG4uQBa2f",
			options:      "w=300,h=300,fit=scale-down",
			webpAccepted: true,
			expectedParams: &ImageParams{
				ImageID:      "NG4uQBa2f",
				Fit:          "scale-down",
				Width:        300,
				Height:       300,
				Quality:      50,
				WebpAccepted: true,
			},
			err: nil,
		},
		{
			testID:       8,
			imageID:      "NG4uQBa2f",
			options:      "w=0,h=0",
			webpAccepted: true,
			expectedParams: &ImageParams{
				ImageID:      "NG4uQBa2f",
				Fit:          "contain",
				Width:        0,
				Height:       0,
				Quality:      50,
				WebpAccepted: true,
			},
			err: nil,
		},
		{
			testID:         9,
			imageID:        "NG4uQBa2f",
			options:        "w=ff,h=0",
			webpAccepted:   true,
			expectedParams: &ImageParams{},
			err:            fmt.Errorf("Width should be integer"),
		},
		{
			testID:         10,
			imageID:        "NG4uQBa2f",
			options:        "w=300,h=gg",
			webpAccepted:   true,
			expectedParams: &ImageParams{},
			err:            fmt.Errorf("Height should be integer"),
		},
		{
			testID:         12,
			imageID:        "NG4uQBa2f",
			options:        "w==",
			webpAccepted:   true,
			expectedParams: &ImageParams{},
			err:            fmt.Errorf("Invalid param: w=="),
		},
		{
			testID:         13,
			imageID:        "NG4uQBa2f",
			options:        "fit=stretch",
			webpAccepted:   true,
			expectedParams: &ImageParams{},
			err:            fmt.Errorf("Supported fits are cover, contain and scale-down"),
		},
		{
			testID:         15,
			imageID:        "NG4uQBa2f",
			options:        "k=k",
			webpAccepted:   true,
			expectedParams: &ImageParams{},
			err:            fmt.Errorf("Invalid filter key: k"),
		},
		{
			testID:       16,
			imageID:      "NG4uQBa2f",
			options:      "q=95",
			webpAccepted: true,
			expectedParams: &ImageParams{
				ImageID:      "NG4uQBa2f",
				Fit:          "contain",
				Width:        0,
				Height:       0,
				Quality:      95,
				WebpAccepted: true,
			},
			err: nil,
		},
		{
			testID:       17,
			imageID:      "NG4uQBa2f",
			options:      "q=m",
			webpAccepted: true,
			expectedParams: &ImageParams{
				ImageID:      "NG4uQBa2f",
				Fit:          "contain",
				Width:        0,
				Height:       0,
				Quality:      95,
				WebpAccepted: true,
			},
			err: fmt.Errorf("Quality should be integer"),
		},
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("ImageParamsFromUri %d", tc.testID), func(t *testing.T) {
			is := is.NewRelaxed(t)
			resultParams, err := createImageParams(
				tc.imageID,
				tc.options,
				tc.webpAccepted,
				config,
			)

			if tc.err != nil {
				is.True(err != nil)
				is.Equal(tc.err.Error(), err.Error())
			} else {
				is.Equal(tc.expectedParams, resultParams)
			}
		})
	}

}

func TestGetParamsToBimgOptions(t *testing.T) {
	tt := []struct {
		name        string
		imageParams *ImageParams
		imageSize   *bimg.ImageSize
		options     *bimg.Options
	}{
		{
			name: "webp_accepted_false",
			imageParams: &ImageParams{
				Width:        300,
				Height:       300,
				Fit:          "cover",
				Quality:      80,
				WebpAccepted: false,
			},
			imageSize: &bimg.ImageSize{
				Width:  900,
				Height: 800,
			},
			options: &bimg.Options{
				Width:  300,
				Height: 300,
				Type:   bimg.JPEG,
				Crop:   true,
				Embed:  true,
			},
		},
		{
			name: "webp_accepted_true",
			imageParams: &ImageParams{
				Width:        300,
				Height:       300,
				Fit:          "cover",
				Quality:      80,
				WebpAccepted: true,
			},
			imageSize: &bimg.ImageSize{
				Width:  900,
				Height: 800,
			},
			options: &bimg.Options{
				Width:  300,
				Height: 300,
				Type:   bimg.WEBP,
				Crop:   true,
				Embed:  true,
			},
		},
		{
			name: "cover_landscape",
			imageParams: &ImageParams{
				Width:        300,
				Height:       300,
				Fit:          "cover",
				Quality:      80,
				WebpAccepted: true,
			},
			imageSize: &bimg.ImageSize{
				Width:  900,
				Height: 400,
			},
			options: &bimg.Options{
				Width:  300,
				Height: 300,
				Type:   bimg.WEBP,
				Crop:   true,
				Embed:  true,
			},
		},
		{
			name: "cover_portrait",
			imageParams: &ImageParams{
				Width:        300,
				Height:       300,
				Fit:          "cover",
				Quality:      80,
				WebpAccepted: true,
			},
			imageSize: &bimg.ImageSize{
				Width:  400,
				Height: 900,
			},
			options: &bimg.Options{
				Width:  300,
				Height: 300,
				Type:   bimg.WEBP,
				Crop:   true,
				Embed:  true,
			},
		},
		{
			name: "contain_landscape_width_restrict",
			imageParams: &ImageParams{
				Width:        300,
				Height:       300,
				Fit:          "contain",
				Quality:      80,
				WebpAccepted: true,
			},
			imageSize: &bimg.ImageSize{
				Width:  900,
				Height: 400,
			},
			options: &bimg.Options{
				Width:  300,
				Height: 0,
				Type:   bimg.WEBP,
				Crop:   false,
				Embed:  false,
			},
		},
		{
			name: "contain_landscape_height_restrict",
			imageParams: &ImageParams{
				Width:        900,
				Height:       300,
				Fit:          "contain",
				Quality:      80,
				WebpAccepted: true,
			},
			imageSize: &bimg.ImageSize{
				Width:  900,
				Height: 400,
			},
			options: &bimg.Options{
				Width:  0,
				Height: 300,
				Type:   bimg.WEBP,
				Crop:   false,
				Embed:  false,
			},
		},
		{
			name: "contain_only_height",
			imageParams: &ImageParams{
				Height:       300,
				Fit:          "contain",
				Quality:      80,
				WebpAccepted: true,
			},
			imageSize: &bimg.ImageSize{
				Width:  900,
				Height: 400,
			},
			options: &bimg.Options{
				Width:  0,
				Height: 300,
				Type:   bimg.WEBP,
				Crop:   false,
				Embed:  false,
			},
		},
		{
			name: "contain_only_width",
			imageParams: &ImageParams{
				Width:        300,
				Fit:          "contain",
				Quality:      80,
				WebpAccepted: true,
			},
			imageSize: &bimg.ImageSize{
				Width:  900,
				Height: 400,
			},
			options: &bimg.Options{
				Width: 300,
				Type:  bimg.WEBP,
				Crop:  false,
				Embed: false,
			},
		},
		{
			name: "scale-down-width-gt-heigh",
			imageParams: &ImageParams{
				Width:        1200,
				Fit:          "scale-down",
				Quality:      80,
				WebpAccepted: true,
			},
			imageSize: &bimg.ImageSize{
				Width:  900,
				Height: 400,
			},
			options: &bimg.Options{
				Width: 900,
				Type:  bimg.WEBP,
				Crop:  false,
				Embed: false,
			},
		},
		{
			name: "scale-down-height-gt-width",
			imageParams: &ImageParams{
				Height:       1200,
				Fit:          "scale-down",
				Quality:      80,
				WebpAccepted: true,
			},
			imageSize: &bimg.ImageSize{
				Width:  400,
				Height: 900,
			},
			options: &bimg.Options{
				Height: 900,
				Type:   bimg.WEBP,
				Crop:   false,
				Embed:  false,
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			opts := tc.imageParams.toBimgOptions(tc.imageSize)
			if !bimgOptsAreEqual(tc.options, opts) {
				t.Fatalf("Expected %s but result is %s",
					bimgOptsToString(tc.options),
					bimgOptsToString(opts),
				)
			}
		})
	}
}
