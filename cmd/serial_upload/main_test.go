// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"io"
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
				Output:     io.Discard,
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

func TestUploadLinger(t *testing.T) {
	ptmx, pts, err := pty.Open()
	if err != nil {
		t.Fatalf("failed to open pty: %v", err)
	}
	// We'll close ptmx explicitly to stop the linger loop.
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

	fileContent := "some content"
	prompt := "PROMPT"

	tmpfile, err := os.CreateTemp("", "upload-linger-test-")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(fileContent)); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	var outBuf bytes.Buffer
	cfg := Config{
		FileName:   tmpfile.Name(),
		DeviceName: pts.Name(),
		Prompt:     prompt,
		BaudRate:   115200,
		StartBits:  8,
		StopBits:   1,
		Parity:     "N",
		Linger:     true,
		Output:     &outBuf,
		Copy:       true,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- upload(cfg)
	}()

	// Wait for startup
	time.Sleep(100 * time.Millisecond)

	// Send prompt
	if _, err := fmt.Fprintln(ptmx, prompt); err != nil {
		t.Fatalf("failed to write prompt: %v", err)
	}

	// Read file content (uploaded)
	buf := make([]byte, 1024)
	n, err := ptmx.Read(buf)
	if err != nil {
		t.Fatalf("failed to read from pty: %v", err)
	}
	if !bytes.Equal(buf[:n], []byte(fileContent)) {
		t.Errorf("got %q, want %q", buf[:n], fileContent)
	}

	// Now send some data back to be lingered
	lingerData := "lingering data"
	if _, err := fmt.Fprintln(ptmx, lingerData); err != nil {
		t.Fatalf("failed to write linger data: %v", err)
	}

	// Wait a bit for data to propagate
	time.Sleep(100 * time.Millisecond)

	// Check if data appeared in outBuf
	// Note: Fprintln adds a newline, and the pty might do newline translation depending on settings,
	// but we disabled ONLCR on master side.
	// We might need to check contains or trim.
	if !bytes.Contains(outBuf.Bytes(), []byte(lingerData)) {
		t.Errorf("output buffer does not contain %q. Got: %q", lingerData, outBuf.String())
	}

	// Close ptmx to stop the linger loop (io.Copy should return error or EOF)
	ptmx.Close()

	// Wait for upload to return
	select {
	case err := <-errCh:
		// We expect an error here because pty was closed, or maybe nil if it handled EOF gracefully.
		// io.Copy usually returns nil on EOF.
		if err != nil {
			t.Logf("upload returned error (expected due to close): %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("upload did not return after pty close")
	}
}
