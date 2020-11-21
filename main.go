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
	handler := &Handler{
		sid: shortid.MustNew(1, shortid.DefaultABC, 535342),
	}
	addr := fmt.Sprintf("127.0.0.1:%d", config.SERVER_PORT)
	log.Printf("Starting server on %s", addr)
	fasthttp.ListenAndServe(addr, handler.handleRequests)
}
