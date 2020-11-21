package main

import (
	"fmt"
	"log"

	"github.com/teris-io/shortid"
	"github.com/valyala/fasthttp"
)

var config *Config

func init() {
	config = GetConfig()
}

func main() {
	handler := &Handler{}
	var err error
	if handler.sid, err = shortid.New(1, shortid.DefaultABC, 2342); err != nil {
		log.Fatalf("Failed createing shortid seed: %v", err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", config.SERVER_PORT)
	log.Printf("Starting server on %s", addr)
	fasthttp.ListenAndServe(addr, handler.handleRequests)
}
