package main

import (
	"flag"
	"fmt"
	bimg "gopkg.in/h2non/bimg.v1"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func checkVipsVersion(majorVersion, minorVersion int) error {
	minMajorVersion, minMinorVersion := 8, 9

	if (majorVersion < minMajorVersion) || (majorVersion == minMajorVersion && minorVersion < minMinorVersion) {
		return fmt.Errorf("Install libips=>'%d.%d'. Current version is %d.%d",
			minMajorVersion, minMinorVersion, majorVersion, minorVersion)
	}
	return nil
}

func main() {
	if err := checkVipsVersion(bimg.VipsMajorVersion, bimg.VipsMinorVersion); err != nil {
		log.Fatal(err)
	}
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
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	server := createServer(config)
	log.Printf("Starting server on %s", config.ServerAddress)
	go func() {
		if err := server.ListenAndServe(config.ServerAddress); err != nil {
			log.Fatal(err)
		}
	}()
	<-done
	if err := server.Shutdown(); err != nil {
		log.Fatal(err)
	}
}
