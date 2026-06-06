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
	lineBuffer = flag.Bool("line-buffer", false, "wait for an XON character to arrive after a single line has been emitted before sending the next line")
	logFlag    = flag.Bool("log", false, "log to stderr all the lines sent")
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
	LineBuffer bool
	Log        bool
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
		LineBuffer: *lineBuffer,
		Log:        *logFlag,
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

type chanReader struct {
	ch    <-chan byte
	errCh <-chan error
	err   error
}

func (r *chanReader) Read(p []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	if len(p) == 0 {
		return 0, nil
	}

	b, ok := <-r.ch
	if !ok {
		select {
		case err := <-r.errCh:
			r.err = err
			if err == io.EOF {
				return 0, io.EOF
			}
			return 0, err
		default:
			return 0, io.EOF
		}
	}
	p[0] = b
	n := 1

	for i := 1; i < len(p); i++ {
		select {
		case b, ok := <-r.ch:
			if !ok {
				return n, nil
			}
			p[i] = b
			n++
		default:
			return n, nil
		}
	}
	return n, nil
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

	byteCh := make(chan byte, 1024*1024)
	errCh := make(chan error, 1)
	pauseCh := make(chan bool, 10)

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := port.Read(buf)
			if n > 0 {
				for i := 0; i < n; i++ {
					b := buf[i]
					if b == 0x13 { // XOFF
						if cfg.Log {
							fmt.Fprintln(os.Stderr, "received: XOFF")
						}
						select {
						case pauseCh <- true:
						default:
						}
					} else if b == 0x11 { // XON
						if cfg.Log {
							fmt.Fprintln(os.Stderr, "received: XON")
						}
						select {
						case pauseCh <- false:
						default:
						}
					} else {
						select {
						case byteCh <- b:
						default:
						}
					}
				}
			}
			if err != nil {
				errCh <- err
				close(byteCh)
				return
			}
		}
	}()

	cr := &chanReader{ch: byteCh, errCh: errCh}
	scanner := bufio.NewScanner(cr)
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

			buf := make([]byte, 64)
			paused := false
			var sendErr error
			br := bufio.NewReader(file)
		SendLoop:
			for {
				for {
					select {
					case p := <-pauseCh:
						paused = p
						continue
					default:
					}
					break
				}

				if paused {
					select {
					case p := <-pauseCh:
						paused = p
					case err := <-errCh:
						sendErr = err
						break SendLoop
					}
					continue
				}

				var toWrite []byte
				var readErr error

				if cfg.LineBuffer {
					toWrite, readErr = br.ReadBytes('\n')
				} else {
					n, err := br.Read(buf)
					if n > 0 {
						toWrite = buf[:n]
					}
					readErr = err
				}

				if len(toWrite) > 0 {
					for len(toWrite) > 0 {
						for {
							select {
							case p := <-pauseCh:
								paused = p
								continue
							default:
							}
							break
						}

						if paused {
							select {
							case p := <-pauseCh:
								paused = p
							case err := <-errCh:
								sendErr = err
								break SendLoop
							}
							continue
						}

						chunkSize := 64
						if len(toWrite) < chunkSize {
							chunkSize = len(toWrite)
						}

						if _, err := port.Write(toWrite[:chunkSize]); err != nil {
							sendErr = fmt.Errorf("failed to write to serial port: %w", err)
							break SendLoop
						}

						if cfg.Log {
							fmt.Fprintf(os.Stderr, "sent: %q\n", string(toWrite[:chunkSize]))
						}

						toWrite = toWrite[chunkSize:]
					}

					if cfg.LineBuffer {
						paused = true
					}
				}

				if readErr == io.EOF {
					break
				}
				if readErr != nil {
					sendErr = fmt.Errorf("failed to read file: %w", readErr)
					break
				}
			}

			if sendErr != nil {
				return sendErr
			}
			fmt.Printf("file sent\n")

			if cfg.Linger {
				fmt.Println("lingering...")
				if cfg.Copy {
					if _, err := io.Copy(cfg.Output, cr); err != nil {
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
