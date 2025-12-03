package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/filmil/futility/seriallib"
)

var (
	fileName    = flag.String("file", "", "file name to upload")
	deviceName  = flag.String("device", "", "serial port device name")
	baudRate    = flag.Int("baud", 115200, "baud rate")
	startBits   = flag.Int("startbits", 8, "start bits")
	stopBits    = flag.Int("stopbits", 1, "stop bits")
	parity      = flag.String("parity", "N", "parity (N, O, E)")
	prompt      = flag.String("prompt", "", "prompt line to wait for")
)

type Config struct {
	FileName   string
	DeviceName string
	BaudRate   int
	StartBits  int
	StopBits   int
	Parity     string
	Prompt     string
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
	}

	if err := upload(cfg); err != nil {
		log.Fatal(err)
	}
}

func upload(cfg Config) error {
	port, err := seriallib.Open(cfg.DeviceName)
	if err != nil {
		return fmt.Errorf("failed to open serial port: %w", err)
	}
	defer port.Close()

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
	fmt.Printf("waiting for prompt %q\n", cfg.Prompt)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("read line: %q\n", line)
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
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading from serial port: %w", err)
	}

	return fmt.Errorf("prompt not found")
}