package main

import (
	"bytes"
	"fmt"
	"log"

	"github.com/valyala/fasthttp"
	bimg "gopkg.in/h2non/bimg.v1"
)

var config *Config

func init() {
	config = GetConfig()
}

func RequestHandler(ctx *fasthttp.RequestCtx) {
	if string(ctx.Path()) == "/health/" {
		ctx.SetStatusCode(200)
		return
	}
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

func main() {
	fasthttp.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", config.SERVER_PORT), RequestHandler)

}
