package main

import (
	"github.com/valyala/fasthttp"
	"log"
	"os"
	"strings"
)

func runServer(config *Config) {
	handler := &Handler{Config: config}
	addr := config.SERVER_ADDRESS
	log.Printf("Starting server on %s", addr)
	server := &fasthttp.Server{
		Handler:               handler.handleRequests,
		NoDefaultServerHeader: true,
		MaxRequestBodySize:    config.MAX_REQUEST_BODY_SIZE * 1024 * 1024,
	}
	var err error

	if strings.HasPrefix(addr, "unix://") {
		err = server.ListenAndServeUNIX(addr, os.ModeSocket)
	} else {
		err = server.ListenAndServe(addr)
	}
	if err != nil {
		log.Println(err)
	}
}

func main() {
	config := GetConfig()
	runServer(config)
}
