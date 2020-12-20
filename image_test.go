package main

import (
	"fmt"
	"github.com/valyala/fasthttp"
	bimg "gopkg.in/h2non/bimg.v1"
	"testing"
)

func createRequestHeader(uri string, accept_webp bool) *fasthttp.RequestHeader {
	header := &fasthttp.RequestHeader{}
	header.SetRequestURI(uri)
	if accept_webp {
		header.SetBytesKV([]byte("accept"), []byte("webp"))
	}
	return header
}

func (p1 *ImageParams) IsEqual(p2 *ImageParams) bool {
	return p1.GetMd5() == p2.GetMd5()
}

func (p *ImageParams) String() string {
	return fmt.Sprintf(
		"id:%s,width:%d,height:%d,format:%s,fit:%s,quality:%d,webp_accepted:%t",
		p.ImageId, p.Width, p.Height, p.Format, p.Fit, p.Quality, p.WebpAccepted,
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
	parentDir, filePath := ImageIdToFilePath("/tmp/media", "FyBmW7C2f")
	if parentDir != "/tmp/media/images/y/mW" {
		t.Errorf("Something wrong with image file parentDir: %s", parentDir)
	}

	if filePath != "/tmp/media/images/y/mW/FyBmW7C2f" {
		t.Errorf("Something wrong with image file path: %s", filePath)
	}
}

func TestCachePath(t *testing.T) {
	params := &ImageParams{
		ImageId:      "NG4uQBa2f",
		Width:        100,
		Height:       100,
		Format:       "auto",
		Fit:          "cover",
		Quality:      90,
		WebpAccepted: true,
	}

	expectedKey := "f3a3ecbb2012b714bdbe7d1e21cf012a"

	if cacheKey := params.GetMd5(); cacheKey != expectedKey {
		t.Errorf("Something wrong with md5: %s", cacheKey)
	}

	parentDir, filePath := params.GetCachePath("/tmp/media/")

	if parentDir != "/tmp/media/caches/a/12" {
		t.Errorf("Something wrong with cache parentDir: %s", parentDir)
	}

	if filePath != fmt.Sprintf("/tmp/media/caches/a/12/%s", expectedKey) {
		t.Errorf("Something wrong with cache file path: %s", filePath)
	}
}

func TestGetParamsFromUri(t *testing.T) {
	config := &Config{
		DATA_DIR:              "/tmp/media/",
		DEFAULT_IMAGE_QUALITY: 50,
	}

	tt := []struct {
		testId         int
		imageId        string
		options        string
		webpAccepted   bool
		expectedParams *ImageParams
		err            error
	}{
		{
			testId:       1,
			imageId:      "NG4uQBa2f",
			options:      "w=500,h=500,fit=contain",
			webpAccepted: false,
			expectedParams: &ImageParams{
				ImageId:      "NG4uQBa2f",
				Fit:          "contain",
				Format:       "auto",
				Width:        500,
				Height:       500,
				Quality:      50,
				WebpAccepted: false,
			},
			err: nil,
		},
		{
			testId:       2,
			imageId:      "NG4uQBa2f",
			options:      "w=300,h=300,fit=contain",
			webpAccepted: false,
			expectedParams: &ImageParams{
				ImageId:      "NG4uQBa2f",
				Fit:          "contain",
				Format:       "auto",
				Width:        300,
				Height:       300,
				Quality:      50,
				WebpAccepted: false,
			},
			err: nil,
		},
		{
			testId:       3,
			imageId:      "NG4uQBa2f",
			options:      "w=300,h=300,fit=contain",
			webpAccepted: true,
			expectedParams: &ImageParams{
				ImageId:      "NG4uQBa2f",
				Fit:          "contain",
				Format:       "auto",
				Width:        300,
				Height:       300,
				Quality:      50,
				WebpAccepted: true,
			},
			err: nil,
		},
		{
			testId:       4,
			imageId:      "NG4uQBa2f",
			options:      "w=300,h=300,fit=cover",
			webpAccepted: true,
			expectedParams: &ImageParams{
				ImageId:      "NG4uQBa2f",
				Fit:          "cover",
				Format:       "auto",
				Width:        300,
				Height:       300,
				Quality:      50,
				WebpAccepted: true,
			},
			err: nil,
		},
		{
			testId:       5,
			imageId:      "NG4uQBa2f",
			options:      "w=300,h=300,fit=cover,format=jpeg",
			webpAccepted: true,
			expectedParams: &ImageParams{
				ImageId:      "NG4uQBa2f",
				Fit:          "cover",
				Format:       "jpeg",
				Width:        300,
				Height:       300,
				Quality:      50,
				WebpAccepted: true,
			},
			err: nil,
		},
		{
			testId:       7,
			imageId:      "NG4uQBa2f",
			options:      "w=300,h=300,fit=scale-down,format=jpeg",
			webpAccepted: true,
			expectedParams: &ImageParams{
				ImageId:      "NG4uQBa2f",
				Fit:          "scale-down",
				Format:       "jpeg",
				Width:        300,
				Height:       300,
				Quality:      50,
				WebpAccepted: true,
			},
			err: nil,
		},
		{
			testId:       8,
			imageId:      "NG4uQBa2f",
			options:      "w=0,h=0,format=auto",
			webpAccepted: true,
			expectedParams: &ImageParams{
				ImageId:      "NG4uQBa2f",
				Fit:          "contain",
				Format:       "auto",
				Width:        0,
				Height:       0,
				Quality:      50,
				WebpAccepted: true,
			},
			err: nil,
		},
		{
			testId:         9,
			imageId:        "NG4uQBa2f",
			options:        "w=ff,h=0,format=jpeg",
			webpAccepted:   true,
			expectedParams: &ImageParams{},
			err:            fmt.Errorf("Width should be integer"),
		},
		{
			testId:         10,
			imageId:        "NG4uQBa2f",
			options:        "w=300,h=gg,format=jpeg",
			webpAccepted:   true,
			expectedParams: &ImageParams{},
			err:            fmt.Errorf("Height should be integer"),
		},
		{
			testId:         12,
			imageId:        "NG4uQBa2f",
			options:        "w==",
			webpAccepted:   true,
			expectedParams: &ImageParams{},
			err:            fmt.Errorf("Invalid param: w=="),
		},
		{
			testId:         13,
			imageId:        "NG4uQBa2f",
			options:        "fit=stretch",
			webpAccepted:   true,
			expectedParams: &ImageParams{},
			err:            fmt.Errorf("Supported fits are cover, contain and scale-down"),
		},
		{
			testId:         14,
			imageId:        "NG4uQBa2f",
			options:        "format=gif",
			webpAccepted:   true,
			expectedParams: &ImageParams{},
			err:            fmt.Errorf("Supported formats are auto, original, webp, jpeg"),
		},
		{
			testId:         15,
			imageId:        "NG4uQBa2f",
			options:        "k=k",
			webpAccepted:   true,
			expectedParams: &ImageParams{},
			err:            fmt.Errorf("Invalid filter key: k"),
		},
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("ImageParamsFromUri %d", tc.testId), func(t *testing.T) {
			resultParams, err := CreateImageParams(
				tc.imageId,
				tc.options,
				tc.webpAccepted,
				config,
			)

			if tc.err != nil {
				if tc.err.Error() != err.Error() {
					t.Fatalf("Expected error %v but got %v", tc.err, err)
				}
			} else {
				if !tc.expectedParams.IsEqual(resultParams) {
					t.Fatalf("Expected %v as imageParams but result is %v",
						tc.expectedParams, resultParams)
				}
			}
		})
	}

}

func TestGetParamsToBimgOptions(t *testing.T) {
	tt := []struct {
		name        string
		imageParams *ImageParams
		imageSize   *bimg.ImageSize
		imageType   bimg.ImageType
		options     *bimg.Options
	}{
		{
			name: "webp_accepted_false",
			imageParams: &ImageParams{
				Width:        300,
				Height:       300,
				Format:       "auto",
				Fit:          "cover",
				Quality:      80,
				WebpAccepted: false,
			},
			imageSize: &bimg.ImageSize{
				Width:  900,
				Height: 800,
			},
			imageType: bimg.PNG,
			options: &bimg.Options{
				Width:  300,
				Height: 300,
				Type:   bimg.JPEG,
				Crop:   true,
				Embed:  true,
			},
		},
		{
			name: "original_image",
			imageParams: &ImageParams{
				Width:        300,
				Height:       300,
				Format:       "original",
				Fit:          "cover",
				Quality:      80,
				WebpAccepted: false,
			},
			imageSize: &bimg.ImageSize{
				Width:  900,
				Height: 800,
			},
			imageType: bimg.PNG,
			options: &bimg.Options{
				Width:  300,
				Height: 300,
				Type:   bimg.PNG,
				Crop:   true,
				Embed:  true,
			},
		},

		{
			name: "webp_accepted_true",
			imageParams: &ImageParams{
				Width:        300,
				Height:       300,
				Format:       "auto",
				Fit:          "cover",
				Quality:      80,
				WebpAccepted: true,
			},
			imageSize: &bimg.ImageSize{
				Width:  900,
				Height: 800,
			},
			imageType: bimg.PNG,
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
				Format:       "auto",
				Fit:          "cover",
				Quality:      80,
				WebpAccepted: true,
			},
			imageSize: &bimg.ImageSize{
				Width:  900,
				Height: 400,
			},
			imageType: bimg.PNG,
			options: &bimg.Options{
				Width:  300,
				Height: 300,
				Type:   bimg.WEBP,
				Crop:   true,
				Embed:  true,
			},
		},
		{
			name: "cover_portait",
			imageParams: &ImageParams{
				Width:        300,
				Height:       300,
				Format:       "auto",
				Fit:          "cover",
				Quality:      80,
				WebpAccepted: true,
			},
			imageSize: &bimg.ImageSize{
				Width:  400,
				Height: 900,
			},
			imageType: bimg.PNG,
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
				Format:       "auto",
				Fit:          "contain",
				Quality:      80,
				WebpAccepted: true,
			},
			imageSize: &bimg.ImageSize{
				Width:  900,
				Height: 400,
			},
			imageType: bimg.PNG,
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
				Format:       "auto",
				Fit:          "contain",
				Quality:      80,
				WebpAccepted: true,
			},
			imageSize: &bimg.ImageSize{
				Width:  900,
				Height: 400,
			},
			imageType: bimg.PNG,
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
				Format:       "auto",
				Fit:          "contain",
				Quality:      80,
				WebpAccepted: true,
			},
			imageSize: &bimg.ImageSize{
				Width:  900,
				Height: 400,
			},
			imageType: bimg.PNG,
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
				Format:       "auto",
				Fit:          "contain",
				Quality:      80,
				WebpAccepted: true,
			},
			imageSize: &bimg.ImageSize{
				Width:  900,
				Height: 400,
			},
			imageType: bimg.PNG,
			options: &bimg.Options{
				Width: 300,
				Type:  bimg.WEBP,
				Crop:  false,
				Embed: false,
			},
		},
		{
			name: "scale-down",
			imageParams: &ImageParams{
				Width:        1200,
				Format:       "auto",
				Fit:          "scale-down",
				Quality:      80,
				WebpAccepted: true,
			},
			imageSize: &bimg.ImageSize{
				Width:  900,
				Height: 400,
			},
			imageType: bimg.PNG,
			options: &bimg.Options{
				Width: 900,
				Type:  bimg.WEBP,
				Crop:  false,
				Embed: false,
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			opts := tc.imageParams.ToBimgOptions(tc.imageSize, tc.imageType)

			if !bimgOptsAreEqual(tc.options, opts) {
				t.Fatalf("Expected %s but result is %s",
					bimgOptsToString(tc.options),
					bimgOptsToString(opts),
				)
			}
		})
	}
}
