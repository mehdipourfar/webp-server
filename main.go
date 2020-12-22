package main

import (
	"flag"
	"github.com/valyala/fasthttp"
	"log"
	"os"
	"strings"
)

func runServer(config *Config) {
	handler := &Handler{Config: config}
	addr := config.ServerAddress
	log.Printf("Starting server on %s", addr)
	server := &fasthttp.Server{
		Handler:               handler.handleRequests,
		NoDefaultServerHeader: true,
		MaxRequestBodySize:    config.MaxRequestBodySize * 1024 * 1024,
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
	configPath := flag.String("config", "", "Path of config file in yml format")
	flag.Parse()
	if *configPath == "" {
		log.Fatal("Set config.yml path via -config flag.")
	}
	config := ParseConfig(*configPath)
	log.Printf("%+v", config)
	runServer(config)
}
