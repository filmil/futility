package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/creack/pty"
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

			tmpfile, err := ioutil.TempFile("", "upload-test-")
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
			*fileName = tmpfile.Name()
			*deviceName = pts.Name()
			*prompt = tt.prompt

			errCh := make(chan error, 1)
			go func() {
				errCh <- upload()
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
