package main

import (
	"fmt"
	"log"

	"github.com/teris-io/shortid"
	"github.com/valyala/fasthttp"
)

func runServer(config *Config) {
	sid := shortid.MustNew(1, shortid.DefaultABC, 535342)
	shortid.SetDefault(sid)
	handler := &Handler{Config: config}
	addr := fmt.Sprintf("127.0.0.1:%d", config.SERVER_PORT)
	log.Printf("Starting server on %s", addr)
	err := fasthttp.ListenAndServe(addr, handler.handleRequests)
	if err != nil {
		log.Println(err)
	}
}

func main() {
	config := GetConfig()
	runServer(config)
}
