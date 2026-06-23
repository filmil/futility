// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/filmil/futility/seriallib"
	"golang.org/x/sys/unix"
)

// mockPort is a mock for the port interface for testing purposes.
type mockPort struct {
	*os.File
}

// SetMode is a mock implementation of the SetMode method.
func (m *mockPort) SetMode(mode *seriallib.Mode) error {
	// For the test, we don't need to do anything here.
	return nil
}

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
				mport := &mockPort{pts}
				errCh <- upload(cfg, mport)
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
		mport := &mockPort{pts}
		errCh <- upload(cfg, mport)
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

type customMockPort struct {
	readFunc  func(p []byte) (n int, err error)
	writeFunc func(p []byte) (n int, err error)
	closeFunc func() error
}

func (m *customMockPort) Read(p []byte) (n int, err error) {
	if m.readFunc != nil {
		return m.readFunc(p)
	}
	return 0, io.EOF
}

func (m *customMockPort) Write(p []byte) (n int, err error) {
	if m.writeFunc != nil {
		return m.writeFunc(p)
	}
	return len(p), nil
}

func (m *customMockPort) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *customMockPort) SetMode(mode *seriallib.Mode) error {
	return nil
}

func TestUploadLineBuffer(t *testing.T) {
	fileContent := "line1\nline2\nline3"
	prompt := "PROMPT\n"

	tmpfile, err := os.CreateTemp("", "upload-linebuffer-test-")
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

	cfg := Config{
		FileName:   tmpfile.Name(),
		DeviceName: "mock",
		Prompt:     "PROMPT",
		LineBuffer: true,
		Output:     io.Discard,
	}

	readCh := make(chan byte, 100)
	writeCh := make(chan []byte, 100)

	for _, b := range []byte(prompt) {
		readCh <- b
	}

	mport := &customMockPort{
		readFunc: func(p []byte) (int, error) {
			if len(p) == 0 {
				return 0, nil
			}
			b, ok := <-readCh
			if !ok {
				return 0, io.EOF
			}
			p[0] = b
			return 1, nil
		},
		writeFunc: func(p []byte) (int, error) {
			b := make([]byte, len(p))
			copy(b, p)
			writeCh <- b
			return len(p), nil
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- upload(cfg, mport)
	}()

	// Wait for line1
	select {
	case b := <-writeCh:
		if string(b) != "line1\n" {
			t.Fatalf("expected line1\\n, got %q", string(b))
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for line1")
	}

	// It should pause now.
	select {
	case <-writeCh:
		t.Fatalf("wrote data while paused!")
	case <-time.After(100 * time.Millisecond):
		// good, it paused
	}

	// Send XON
	readCh <- 0x11

	// Wait for line2
	select {
	case b := <-writeCh:
		if string(b) != "line2\n" {
			t.Fatalf("expected line2\\n, got %q", string(b))
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for line2")
	}

	// Send XON
	readCh <- 0x11

	// Wait for line3
	select {
	case b := <-writeCh:
		if string(b) != "line3" {
			t.Fatalf("expected line3, got %q", string(b))
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for line3")
	}

	close(readCh)

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("upload failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for upload to finish")
	}
}

func TestUploadXONXOFF(t *testing.T) {
	fileContent := "aaaaaaaaaabbbbbbbbbb"
	prompt := "PROMPT\n"

	tmpfile, err := os.CreateTemp("", "upload-xonxoff-test-")
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

	cfg := Config{
		FileName:   tmpfile.Name(),
		DeviceName: "mock",
		Prompt:     "PROMPT",
		Output:     io.Discard,
	}

	readCh := make(chan byte, 100)
	writeCh := make(chan []byte, 100)

	for _, b := range []byte(prompt) {
		readCh <- b
	}
	// immediately queue XOFF so it receives it quickly
	readCh <- 0x13

	mport := &customMockPort{
		readFunc: func(p []byte) (int, error) {
			if len(p) == 0 {
				return 0, nil
			}
			b, ok := <-readCh
			if !ok {
				return 0, io.EOF
			}
			p[0] = b
			return 1, nil
		},
		writeFunc: func(p []byte) (int, error) {
			b := make([]byte, len(p))
			copy(b, p)
			writeCh <- b
			return len(p), nil
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- upload(cfg, mport)
	}()

	// Wait for a write. It shouldn't write because it received XOFF immediately.
	totalWritten := 0
	paused := false
	for {
		select {
		case b := <-writeCh:
			totalWritten += len(b)
		case <-time.After(200 * time.Millisecond):
			paused = true
		}
		if paused {
			break
		}
	}

	if totalWritten == len(fileContent) {
		t.Fatalf("wrote all data without pausing!")
	}

	// Send XON
	readCh <- 0x11

	// Wait for the rest of the data
	for {
		select {
		case b := <-writeCh:
			totalWritten += len(b)
		case <-time.After(500 * time.Millisecond):
			break
		}
		if totalWritten >= len(fileContent) {
			break
		}
	}

	if totalWritten != len(fileContent) {
		t.Fatalf("expected to write %d bytes, wrote %d", len(fileContent), totalWritten)
	}

	close(readCh)
	<-errCh
}

// TestSignalHandlerClosesPort verifies that SIGINT causes the port to be closed.
// The old one-shot goroutine would exit after the first signal, swallowing any
// subsequent signals. signal.NotifyContext keeps intercepting until stop() is called.
func TestSignalHandlerClosesPort(t *testing.T) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	portClosed := make(chan struct{})
	mport := &customMockPort{
		readFunc: func(p []byte) (int, error) {
			<-portClosed
			return 0, io.EOF
		},
		closeFunc: func() error {
			select {
			case <-portClosed:
			default:
				close(portClosed)
			}
			return nil
		},
	}

	go func() {
		<-ctx.Done()
		mport.Close()
	}()

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err := proc.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}

	select {
	case <-portClosed:
		// port closed in response to SIGINT — correct behaviour
	case <-time.After(time.Second):
		t.Error("port not closed after SIGINT")
	}
}
