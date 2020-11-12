package main

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/teris-io/shortid"
	"github.com/valyala/fasthttp"
	bimg "gopkg.in/h2non/bimg.v1"
)

type Handler struct {
	sid *shortid.Shortid
}

func (handler *Handler) handleRequests(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())
	if path == "/health/" && ctx.IsGet() {
		ctx.SetStatusCode(200)
	} else if path == "/upload/" && ctx.IsPost() {
		handleUpload(ctx, handler)
	} else if strings.HasPrefix(path, "/image/") && ctx.IsGet() {
		handleGet(ctx)
	} else {
		ctx.Error("Not Found", 404)
	}
}

func handleGet(ctx *fasthttp.RequestCtx) {
	params, err := GetParamsFromUri(ctx.RequestURI())
	if err != nil {
		ctx.Error("Unsupported Path", 400)
		return
	}
	accept := ctx.Request.Header.Peek("accept")
	params.WebpAccepted = bytes.Contains(accept, []byte("webp"))

	convertedImage, imageType, err := Convert(params)
	if err != nil {
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

func handleUpload(ctx *fasthttp.RequestCtx, handler *Handler) {
	imageId, err := handler.sid.Generate()
	if err != nil {
		ctx.Error("Internal Server Error", 500)
		return
	}

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

	filePath := fmt.Sprintf("%s/%s", config.IMAGES_ROOT, imageId)
	fasthttp.SaveMultipartFile(header, filePath)
	ctx.SetBody([]byte(fmt.Sprintf(`{"image_id": "%s"}`, imageId)))
}
