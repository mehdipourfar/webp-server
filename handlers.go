package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/teris-io/shortid"
	"github.com/valyala/fasthttp"
	bimg "gopkg.in/h2non/bimg.v1"
)

type Handler struct {
	Config *Config
}

func (handler *Handler) handleRequests(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())
	if path == "/health/" && ctx.IsGet() {
		ctx.SetStatusCode(200)
	} else if path == "/upload/" && ctx.IsPost() {
		handler.handleUpload(ctx)
	} else if strings.HasPrefix(path, "/image/") && ctx.IsGet() {
		handler.handleGet(ctx)
	} else {
		ctx.Error("Not Found", 404)
	}
}

func (handler *Handler) handleGet(ctx *fasthttp.RequestCtx) {
	params, err := GetImageParamsFromRequest(&ctx.Request.Header, handler.Config)
	if err != nil {
		log.Println(err)
		ctx.Error("Unsupported Path", 400)
		return
	}
	cachedParentDir, cachedFilePath := params.GetCachePath(handler.Config.DATA_DIR)
	if _, err := os.Stat(cachedFilePath); err == nil && false {
		// cachedFile exists
		fasthttp.ServeFileUncompressed(ctx, cachedFilePath)
		return
	}

	_, imageFilePath := ImageIdToFilePath(handler.Config.DATA_DIR, params.ImageId)

	imgBuffer, err := bimg.Read(imageFilePath)

	if err != nil {
		log.Println(err)
		ctx.Error("Internal Server Error", 500)
		return
	}

	convertedImage, imageType, err := Convert(imgBuffer, params)

	if err != nil {
		if os.IsNotExist(err) {
			ctx.SetStatusCode(404)
			ctx.SetBody([]byte("Not Found"))
			return
		}
		log.Println(err)
		ctx.Error("Internal Server Error", 500)
		return
	}

	if err := os.MkdirAll(cachedParentDir, 0755); err != nil {
		log.Println(err)
		ctx.Error("Internal Server Error", 500)
		return
	}
	if err := ioutil.WriteFile(cachedFilePath, convertedImage, 0604); err != nil {
		log.Println(err)
		ctx.Error("Internal Server Error", 500)
	}

	switch imageType {
	case bimg.JPEG:
		ctx.SetContentType("image/jpeg")
	case bimg.PNG:
		ctx.SetContentType("image/png")
	case bimg.WEBP:
		ctx.SetContentType("image/webp")
	case bimg.GIF:
		ctx.SetContentType("image/gif")
	}
	ctx.SetBody(convertedImage)

}

func (handler *Handler) handleUpload(ctx *fasthttp.RequestCtx) {
	if len(handler.Config.TOKEN) != 0 &&
		handler.Config.TOKEN != string(ctx.Request.Header.Peek("Token")) {
		ctx.SetContentType("application/json")
		ctx.SetBody([]byte(`{"error": "Invalid Token"}`))
		ctx.SetStatusCode(401)
		return
	}

	imageId := shortid.GetDefault().MustGenerate()

	ctx.SetContentType("application/json")
	header, err := ctx.FormFile("image_file")
	if err != nil {
		ctx.SetBody([]byte(`{"error": "image_file field not provided"}`))
		ctx.SetStatusCode(400)
		return
	}
	if imageValidated := ValidateImage(header); !imageValidated {
		ctx.SetBody([]byte(`{"error": "provided file is not an image"}`))
		ctx.SetStatusCode(400)
		return
	}

	parentDir, filePath := ImageIdToFilePath(handler.Config.DATA_DIR, imageId)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		log.Println(err)
		ctx.Error("Internal Server Error", 500)
		return
	}
	if err := fasthttp.SaveMultipartFile(header, filePath); err != nil {
		log.Println(err)
		ctx.Error("Internal Server Error", 500)
		return
	}
	ctx.SetBody([]byte(fmt.Sprintf(`{"image_id": "%s"}`, imageId)))
}
