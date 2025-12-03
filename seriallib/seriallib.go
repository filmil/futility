package seriallib

import (
	"fmt"
	"io"
	"os"
)

// Port is an interface for a serial port.
type Port interface {
	io.ReadWriteCloser
	SetMode(mode *Mode) error
}

// Mode represents the serial port settings.
type Mode struct {
	BaudRate  int
	DataBits  int
	StopBits  int
	Parity    Parity
}

// Parity is the parity setting for a serial port.
type Parity byte

const (
	// ParityNone is no parity.
	ParityNone Parity = 'N'
	// ParityOdd is odd parity.
	ParityOdd Parity = 'O'
	// ParityEven is even parity.
	ParityEven Parity = 'E'
)
func Open(deviceName string) (Port, error) {
	f, err := os.OpenFile(deviceName, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open device %q: %w", deviceName, err)
	}
	return &port{f: f}, nil
}

type port struct {
	f *os.File
}

func (p *port) Read(b []byte) (int, error) {
	return p.f.Read(b)
}

func (p *port) Write(b []byte) (int, error) {
	return p.f.Write(b)
}

func (p *port) Close() error {
	return p.f.Close()
}

func (p *port) SetMode(mode *Mode) error {
	// This is a dummy implementation.
	// In a real scenario, this would configure the serial port.
	return nil
}
