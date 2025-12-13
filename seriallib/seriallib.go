// SPDX-License-Identifier: Apache-2.0

package seriallib

import (
	"fmt"
	"io"

	"go.bug.st/serial"
)

// Port is an interface for a serial port.
type Port interface {
	io.ReadWriteCloser
	SetMode(mode *Mode) error
}

// Mode represents the serial port settings.
type Mode struct {
	BaudRate int
	DataBits int
	StopBits int
	Parity   Parity
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
	p, err := serial.Open(deviceName, &serial.Mode{})
	if err != nil {
		return nil, fmt.Errorf("failed to open device %q: %w", deviceName, err)
	}
	return &port{p: p}, nil
}

type port struct {
	p serial.Port
}

func (p *port) Read(b []byte) (int, error) {
	return p.p.Read(b)
}

func (p *port) Write(b []byte) (int, error) {
	return p.p.Write(b)
}

func (p *port) Close() error {
	return p.p.Close()
}

func (p *port) SetMode(mode *Mode) error {
	var parity serial.Parity
	switch mode.Parity {
	case ParityNone:
		parity = serial.NoParity
	case ParityOdd:
		parity = serial.OddParity
	case ParityEven:
		parity = serial.EvenParity
	default:
		return fmt.Errorf("unknown parity: %c", mode.Parity)
	}

	var stopBits serial.StopBits
	switch mode.StopBits {
	case 1:
		stopBits = serial.OneStopBit
	case 2:
		stopBits = serial.TwoStopBits
	default:
		// go-serial also supports 1.5, but our Mode doesn't represent it.
		return fmt.Errorf("unsupported stop bits: %d", mode.StopBits)
	}

	err := p.p.SetMode(&serial.Mode{
		BaudRate: mode.BaudRate,
		DataBits: mode.DataBits,
		Parity:   parity,
		StopBits: stopBits,
	})
	if err != nil {
		return fmt.Errorf("failed to set mode: %w", err)
	}
	return nil
}
