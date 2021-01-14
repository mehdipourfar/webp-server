package main

import (
	"context"
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

func runServer(ctx context.Context) error {
	if err := checkVipsVersion(bimg.VipsMajorVersion, bimg.VipsMinorVersion); err != nil {
		return err
	}
	configPath := flag.String("config", "", "Path of config file in yml format")
	flag.Parse()
	if *configPath == "" {
		return fmt.Errorf("Set config.yml path via -config flag.")
	}
	file, err := os.Open(*configPath)
	if err != nil {
		return fmt.Errorf("Error loading config: %v", err)
	}
	config, err := parseConfig(file)
	file.Close()
	if err != nil {
		return err
	}
	if config.LogPath != "" {
		logFile, err := os.OpenFile(config.LogPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			return fmt.Errorf("Could not open log file: %v", err)
		}
		defer logFile.Close()
		log.SetOutput(logFile)
	} else {
		log.SetOutput(os.Stdout)
	}

	server := createServer(config)

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	defer close(done)

	serverErr := make(chan error)
	defer close(serverErr)

	go func() {
		log.Printf("Starting server on %s", config.ServerAddress)
		if err := server.ListenAndServe(config.ServerAddress); err != nil {
			serverErr <- err
		}
	}()

	select {
	case <-done:
		return server.Shutdown()
	case <-ctx.Done():
		return server.Shutdown()
	case err := <-serverErr:
		return err
	}
}

func main() {
	ctx := context.Background()
	err := runServer(ctx)
	if err != nil {
		log.Fatal(err)
		ctx.Done()
	}

}
