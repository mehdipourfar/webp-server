package main

import (
	"github.com/valyala/fasthttp"
)

func CreateServer(config *Config) *fasthttp.Server {
	handler := &Handler{Config: config}
	return &fasthttp.Server{
		Handler:               handler.handleRequests,
		ErrorHandler:          handler.handleErrors,
		NoDefaultServerHeader: true,
		MaxRequestBodySize:    config.MaxUploadedImageSize * 1024 * 1024,
	}
}
