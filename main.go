package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func runServer(config *Config) {
	server := CreateServer(config)

	log.Printf("Starting server on %s", config.ServerAddress)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		var err error
		if strings.HasPrefix(config.ServerAddress, "unix:") {
			socketPath := strings.Replace(config.ServerAddress, "unix:", "", -1)
			defer os.Remove(socketPath)
			err = server.ListenAndServeUNIX(socketPath, os.ModeSocket|0666)
		} else {
			err = server.ListenAndServe(config.ServerAddress)
		}
		if err != nil {
			log.Fatalf("Listen error: %v", err)
		}
	}()
	<-done
	log.Print("Graceful Shutdown")
	if err := server.Shutdown(); err != nil {
		log.Fatal(err)
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
	config, err := ParseConfig(file)
	file.Close()
	if err != nil {
		log.Fatal(err)
	}

	if config.LogPath != "" {
		logFile, err := os.OpenFile(config.LogPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Fatalf("Could not open log file: %v", err)
		}
		defer logFile.Close()
		log.SetOutput(logFile)
	} else {
		log.SetOutput(os.Stdout)
	}
	runServer(config)
}
