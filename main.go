package main

import (
	"flag"
	"log"
	"os"
	"strings"
)

func runServer(config *Config) {
	server := CreateServer(config)

	log.Printf("Starting server on %s", config.ServerAddress)

	var err error
	if strings.HasPrefix(config.ServerAddress, "unix://") {
		err = server.ListenAndServeUNIX(config.ServerAddress, os.ModeSocket)
	} else {
		err = server.ListenAndServe(config.ServerAddress)
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
	file, err := os.Open(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	config := ParseConfig(file)
	file.Close()
	log.Printf("%+v", config)
	runServer(config)
}
