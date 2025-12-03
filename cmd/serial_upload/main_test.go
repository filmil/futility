package main

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

func TestUpload(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		prompt      string
	}{
		{
			name:        "simple test",
			fileContent: "hello world",
			prompt:      "GO",
		},
		{
			name:        "multiline test",
			fileContent: "hello\nworld",
			prompt:      "READY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptmx, pts, err := pty.Open()
			if err != nil {
				t.Fatalf("failed to open pty: %v", err)
			}
			defer ptmx.Close()
			defer pts.Close()

			// Disable echoing on the master pty.
			termios, err := unix.IoctlGetTermios(int(ptmx.Fd()), unix.TCGETS)
			if err != nil {
				t.Fatalf("failed to get terminal attributes: %v", err)
			}
			termios.Lflag &^= unix.ECHO
			termios.Oflag &^= unix.ONLCR
			if err := unix.IoctlSetTermios(int(ptmx.Fd()), unix.TCSETS, termios); err != nil {
				t.Fatalf("failed to set terminal attributes: %v", err)
			}

			tmpfile, err := os.CreateTemp("", "upload-test-")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer os.Remove(tmpfile.Name())
			if _, err := tmpfile.Write([]byte(tt.fileContent)); err != nil {
				t.Fatalf("failed to write to temp file: %v", err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatalf("failed to close temp file: %v", err)
			}
			
			cfg := Config{
				FileName:   tmpfile.Name(),
				DeviceName: pts.Name(),
				Prompt:     tt.prompt,
				BaudRate:   115200,
				StartBits:  8,
				StopBits:   1,
				Parity:     "N",
			}

			errCh := make(chan error, 1)
			go func() {
				errCh <- upload(cfg)
			}()

			// Give the program a moment to start up and wait for the prompt.
			time.Sleep(100 * time.Millisecond)

			if _, err := fmt.Fprintln(ptmx, tt.prompt); err != nil {
				t.Fatalf("failed to write prompt to pty: %v", err)
			}

			buf := make([]byte, 1024)
			n, err := ptmx.Read(buf)
			if err != nil {
				t.Fatalf("failed to read from pty: %v", err)
			}

			if !bytes.Equal(buf[:n], []byte(tt.fileContent)) {
				t.Errorf("got %q, want %q", buf[:n], tt.fileContent)
			}

			select {
			case err := <-errCh:
				if err != nil {
					t.Errorf("upload function returned an error: %v", err)
				}
			case <-time.After(1 * time.Second):
				t.Error("upload function timed out")
			}
		})
	}
}
