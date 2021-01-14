package main

import (
	"flag"
	"fmt"
	bimg "gopkg.in/h2non/bimg.v1"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func runServer(config *Config) {
	server := createServer(config)

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
	log.Println("Graceful Shutdown")
	if err := server.Shutdown(); err != nil {
		log.Fatal(err)
	}
}

func checkVipsVersion() {
	minVer := []int{8, 9}
	curVer := []int{bimg.VipsMajorVersion, bimg.VipsMinorVersion}
	errMsg := fmt.Sprintf("Install libips=>'%d.%d'. Current version is %s",
		minVer[0], minVer[1], bimg.VipsVersion)
	if curVer[0] < minVer[0] {
		log.Fatal(errMsg)
	} else if curVer[0] == minVer[0] && curVer[1] < minVer[1] {
		log.Fatal(errMsg)
	}
}

func main() {
	checkVipsVersion()
	configPath := flag.String("config", "", "Path of config file in yml format")
	flag.Parse()
	if *configPath == "" {
		log.Fatal("Set config.yml path via -config flag.")
	}
	file, err := os.Open(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	config, err := parseConfig(file)
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
