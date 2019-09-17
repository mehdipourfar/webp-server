package main

import (
	"fmt"
	"github.com/valyala/fasthttp"
)

var config *Config

func init() {
	config = GetConfig()
}

func RequestHandler(ctx *fasthttp.RequestCtx) {
	params, err := GetParamsFromUri(ctx.RequestURI())
	if err != nil {
		ctx.Error("Unsupported Path", 400)
		return
	}
	convertedImage, err := Convert(params)
	if err != nil {
		ctx.Error("Internal Server Error", 500)
	}
	if params.Webp {
		ctx.SetContentType("image/webp")
	} else {
		ctx.SetContentType("image/jpeg")
	}
	ctx.SetBody(convertedImage)
}

func main() {
	fasthttp.ListenAndServe(fmt.Sprintf(":%d", config.SERVER_PORT), RequestHandler)

}
