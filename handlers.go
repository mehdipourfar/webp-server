package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/teris-io/shortid"
	"github.com/valyala/fasthttp"
	bimg "gopkg.in/h2non/bimg.v1"
)

type Handler struct {
	Config *Config
}

var (
	CT_JPEG = "image/jpeg"
	CT_PNG  = "image/png"
	CT_WEBP = "image/webp"
	CT_GIF  = "image/gif"

	PATH_HEALTH = []byte("/health/")
	PATH_UPLOAD = []byte("/upload/")
	PATH_IMAGE  = []byte("/image/")

	ERROR_METHOD_NOT_ALLOWED = []byte(`{"error": "Method not allowed"}`)
	ERROR_IMAGE_NOT_PROVIDED = []byte(`{"error": "image_file field not provided"}`)
	ERROR_FILE_IS_NOT_IMAGE  = []byte(`{"error": "Provided file is not an accepted image"}`)
	ERROR_INVALID_TOKEN      = []byte(`{"error": "Invalid Token"}`)
	ERROR_INVALID_IMAGE_SIZE = []byte(`{"error": "Invalid image size"}`)
	ERROR_IMAGE_NOT_FOUND    = []byte(`{"error": "Image not found"}`)
	ERROR_ADDRESS_NOT_FOUND  = []byte(`{"error": "Address not found"}`)
	ERROR_SERVER             = []byte(`{"error": "Internal Server Error"}`)
)

func jsonResponse(ctx *fasthttp.RequestCtx, status int, body []byte) {
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	ctx.SetBody(body)
}

func handleError(ctx *fasthttp.RequestCtx) {
	if err := recover(); err != nil {
		ctx.ResetBody()
		jsonResponse(ctx, 500, ERROR_SERVER)
		log.Println(err)
	}
}

func (handler *Handler) handleRequests(ctx *fasthttp.RequestCtx) {
	defer handleError(ctx)

	path := ctx.Path()

	if bytes.HasPrefix(path, PATH_IMAGE) {
		handler.handleFetch(ctx)
	} else if bytes.Equal(path, PATH_UPLOAD) {
		handler.handleUpload(ctx)
	} else if bytes.Equal(path, PATH_HEALTH) {
		jsonResponse(ctx, 200, []byte(`{"status": "ok"}`))
	} else {
		jsonResponse(ctx, 404, ERROR_ADDRESS_NOT_FOUND)
	}
}

func (handler *Handler) handleUpload(ctx *fasthttp.RequestCtx) {
	if !ctx.IsPost() {
		jsonResponse(ctx, 405, ERROR_METHOD_NOT_ALLOWED)
		return
	}

	if len(handler.Config.TOKEN) != 0 &&
		handler.Config.TOKEN != string(ctx.Request.Header.Peek("Token")) {
		jsonResponse(ctx, 401, ERROR_INVALID_TOKEN)
		return
	}

	imageId := shortid.GetDefault().MustGenerate()
	fileHeader, err := ctx.FormFile("image_file")
	if err != nil {
		jsonResponse(ctx, 400, ERROR_IMAGE_NOT_PROVIDED)
		return
	}
	if imageValidated := ValidateImage(fileHeader); !imageValidated {
		jsonResponse(ctx, 400, ERROR_FILE_IS_NOT_IMAGE)
		return
	}

	parentDir, filePath := ImageIdToFilePath(handler.Config.DATA_DIR, imageId)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		panic(err)
		return
	}
	if err := fasthttp.SaveMultipartFile(fileHeader, filePath); err != nil {
		panic(err)
		return
	}
	jsonResponse(ctx, 200, []byte(fmt.Sprintf(`{"image_id": "%s"}`, imageId)))
}

func (handler *Handler) handleFetch(ctx *fasthttp.RequestCtx) {
	if !ctx.IsGet() {
		jsonResponse(ctx, 405, ERROR_METHOD_NOT_ALLOWED)
		return
	}

	imageParams, err := GetImageParamsFromRequest(
		&ctx.Request.Header,
		handler.Config,
	)
	if err != nil {
		errorBody := []byte(fmt.Sprintf(`{"error": "Invalid options: %v"}`, err))
		jsonResponse(ctx, 400, errorBody)
		return
	}
	cacheParentDir, cacheFilePath := imageParams.GetCachePath(handler.Config.DATA_DIR)
	if _, err := os.Stat(cacheFilePath); err == nil {
		// cached file exists
		fasthttp.ServeFileUncompressed(ctx, cacheFilePath)
		return
	}

	if !ValidateImageSize(imageParams.Width, imageParams.Height, handler.Config) {
		jsonResponse(ctx, 403, ERROR_INVALID_IMAGE_SIZE)
		return
	}
	_, imageFilePath := ImageIdToFilePath(handler.Config.DATA_DIR, imageParams.ImageId)

	imgBuffer, err := bimg.Read(imageFilePath)

	if err != nil {
		if os.IsNotExist(err) {
			jsonResponse(ctx, 404, ERROR_IMAGE_NOT_FOUND)
			return
		}
		panic(err)
	}

	convertedImage, imageType, err := Convert(imgBuffer, imageParams)
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(cacheParentDir, 0755); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(cacheFilePath, convertedImage, 0604); err != nil {
		panic(err)
	}

	switch imageType {
	case bimg.JPEG:
		ctx.SetContentType(CT_JPEG)
	case bimg.PNG:
		ctx.SetContentType(CT_PNG)
	case bimg.WEBP:
		ctx.SetContentType(CT_WEBP)
	case bimg.GIF:
		ctx.SetContentType(CT_GIF)
	}
	ctx.SetBody(convertedImage)
}
