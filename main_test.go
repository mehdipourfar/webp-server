package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/matryer/is"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestCheckVersion(t *testing.T) {
	tt := []struct {
		name     string
		majorVer int
		minorVer int
		err      error
	}{
		{
			name:     "lower major version",
			majorVer: 7,
			minorVer: 2,
			err:      fmt.Errorf("Install libips=>'8.9'. Current version is 7.2"),
		},
		{
			name:     "lower minor version",
			majorVer: 8,
			minorVer: 5,
			err:      fmt.Errorf("Install libips=>'8.9'. Current version is 8.5"),
		},
		{
			name:     "equal version",
			majorVer: 8,
			minorVer: 9,
			err:      nil,
		},
		{
			name:     "higher version",
			majorVer: 9,
			minorVer: 2,
			err:      nil,
		},
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("TestCheckVersionFunction: %s", tc.name), func(t *testing.T) {
			is := is.NewRelaxed(t)
			err := checkVipsVersion(tc.majorVer, tc.minorVer)
			is.Equal(err, tc.err)
		})
	}
}

func TestRunServerInvalidConfigFlag(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	time.AfterFunc(50*time.Millisecond, func() {
		cancel()
	})
	err := runServer(ctx)
	is.Equal(err, fmt.Errorf("Set config.yml path via -config flag."))
	flag.CommandLine = flag.NewFlagSet("config", flag.ExitOnError)
	configPath := "/tmp/webpserver_test.yaml"
	os.Remove(configPath)
	os.Args = []string{"webp-server", "-config", configPath}
	err = runServer(ctx)
	is.Equal(err, fmt.Errorf("Error loading config: open /tmp/webpserver_test.yaml: no such file or directory"))
}

func TestRunServerInvalidLogPath(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	time.AfterFunc(50*time.Millisecond, func() {
		cancel()
	})

	configData := []byte(`
data_directory:
  /tmp/wstest/
log_path:
  /tmp/wstest/sfsaf/fsfs
`)
	configPath := "/tmp/webpserver_test.yaml"
	err := ioutil.WriteFile(configPath, configData, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(configPath)
	defer os.RemoveAll("/tmp/wstest/")
	flag.CommandLine = flag.NewFlagSet("config", flag.ExitOnError)
	os.Args = []string{"webp-server", "-config", configPath}
	err = runServer(ctx)
	is.Equal(err.Error(), "Could not open log file: open /tmp/wstest/sfsaf/fsfs: no such file or directory")
}

func TestRunServerGracefulShutdown(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	configData := []byte(`
data_directory:
  /tmp/wstest/
log_path:
  /tmp/wstest/wpl.log
`)
	configPath := "/tmp/webpserver_test.yaml"
	err := ioutil.WriteFile(configPath, configData, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(configPath)
	defer os.RemoveAll("/tmp/wstest/")
	flag.CommandLine = flag.NewFlagSet("config", flag.ExitOnError)
	os.Args = []string{"webp-server", "-config", configPath}
	err = ioutil.WriteFile(configPath, configData, 0644)
	if err != nil {
		t.Fatal(err)
	}
	proc, _ := os.FindProcess(os.Getpid())
	time.AfterFunc(200*time.Millisecond, func() {
		err := proc.Signal(os.Interrupt)
		if err != nil {
			t.Fatal(err)
		}
	})
	err = runServer(ctx)
	is.NoErr(err)
}

func TestRunServerInvalidAddress(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	time.AfterFunc(50*time.Millisecond, func() {
		cancel()
	})
	configData := []byte(`
server_address: nfsfs
data_directory:
  /tmp/wstest/
log_path:
  /tmp/wstest/wpl.log
`)
	configPath := "/tmp/webpserver_test.yaml"
	err := ioutil.WriteFile(configPath, configData, 0644)

	defer os.RemoveAll("/tmp/wstest")
	defer os.Remove(configPath)
	if err != nil {
		t.Fatal(err)
	}
	flag.CommandLine = flag.NewFlagSet("config", flag.ExitOnError)
	os.Args = []string{"webp-server", "-config", configPath}
	err = runServer(ctx)
	is.Equal(err.Error(), "listen tcp4: address nfsfs: missing port in address")

}
