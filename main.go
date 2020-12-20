package main

import (
	"fmt"
	"log"

	"github.com/valyala/fasthttp"
)

func runServer(config *Config) {
	handler := &Handler{Config: config}
	addr := fmt.Sprintf("%s:%d", config.SERVER_ADDR, config.SERVER_PORT)
	log.Printf("Starting server on %s", addr)
	server := &fasthttp.Server{
		Handler:               handler.handleRequests,
		NoDefaultServerHeader: true,
	}
	err := server.ListenAndServe(addr)
	if err != nil {
		log.Println(err)
	}
}

func main() {
	config := GetConfig()
	runServer(config)
}
