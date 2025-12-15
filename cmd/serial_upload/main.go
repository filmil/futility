// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"

	"github.com/filmil/futility/seriallib"
)

var (
	fileName   = flag.String("file", "", "file name to upload")
	deviceName = flag.String("device", "", "serial port device name")
	baudRate   = flag.Int("baud", 115200, "baud rate")
	startBits  = flag.Int("startbits", 8, "start bits")
	stopBits   = flag.Int("stopbits", 1, "stop bits")
	parity     = flag.String("parity", "N", "parity (N, O, E)")
	prompt     = flag.String("prompt", "", "prompt line to wait for")
	linger     = flag.Bool("linger", false, "linger after upload and echo serial output to stdout")
)

type Config struct {
	FileName   string
	DeviceName string
	BaudRate   int
	StartBits  int
	StopBits   int
	Parity     string
	Prompt     string
	Linger     bool
	Output     io.Writer
	Copy       bool
}

// port is an interface that represents a serial port.
// It is used to abstract the real serial port implementation for testing.
type port interface {
	io.ReadWriteCloser
	SetMode(mode *seriallib.Mode) error
}

func main() {
	flag.Parse()

	if *fileName == "" {
		log.Fatal("-file is required")
	}
	if *deviceName == "" {
		log.Fatal("-device is required")
	}
	if *prompt == "" {
		log.Fatal("-prompt is required")
	}

	cfg := Config{
		FileName:   *fileName,
		DeviceName: *deviceName,
		BaudRate:   *baudRate,
		StartBits:  *startBits,
		StopBits:   *stopBits,
		Parity:     *parity,
		Prompt:     *prompt,
		Linger:     *linger,
		Output:     os.Stdout,
	}

	port, err := seriallib.Open(cfg.DeviceName)
	if err != nil {
		log.Fatalf("failed to open serial port: %v", err)
	}
	defer port.Close()

	// Set up channel for signal notifications.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	// Goroutine to handle signals.
	go func() {
		sig := <-sigCh
		fmt.Printf("\nCaught signal (%v), closing serial port.\n", sig)
		// Closing the port will cause any blocked reads to fail, allowing graceful shutdown.
		port.Close()
	}()

	if err := upload(cfg, port); err != nil {
		// When the port is closed by the signal handler, upload will return an error.
		// We log it and exit, which is reasonable behavior.
		log.Fatal(err)
	}
}

func upload(cfg Config, port port) error {
	p := seriallib.ParityNone
	switch cfg.Parity {
	case "O":
		p = seriallib.ParityOdd
	case "E":
		p = seriallib.ParityEven
	}

	if err := port.SetMode(&seriallib.Mode{
		BaudRate: cfg.BaudRate,
		DataBits: cfg.StartBits,
		StopBits: cfg.StopBits,
		Parity:   p,
	}); err != nil {
		return fmt.Errorf("failed to set serial port mode: %w", err)
	}

	scanner := bufio.NewScanner(port)
	prompt := true
	for scanner.Scan() {
		if prompt {
			fmt.Printf("waiting for prompt %q\n", cfg.Prompt)
			prompt = false
		}
		line := scanner.Text()
		fmt.Printf("> %q\n", line)
		if line == cfg.Prompt {
			fmt.Printf("prompt received, sending file\n")
			file, err := os.Open(cfg.FileName)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer file.Close()

			if _, err := io.Copy(port, file); err != nil {
				return fmt.Errorf("failed to write file to serial port: %w", err)
			}
			fmt.Printf("file sent\n")

			if cfg.Linger {
				fmt.Println("lingering...")
				if cfg.Copy {
					if _, err := io.Copy(cfg.Output, port); err != nil {
						return fmt.Errorf("error lingering: %w", err)
					}
				}
				prompt = true
			} else {
				fmt.Println("done")
				return nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading from serial port: %w", err)
	}

	return fmt.Errorf("prompt not found")
}
