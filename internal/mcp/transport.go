package mcp

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
)

// StdioTransport implements Transport using stdin/stdout
type StdioTransport struct {
	reader *bufio.Reader
	writer *bufio.Writer
	mu     sync.Mutex
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport() Transport {
	return &StdioTransport{
		reader: bufio.NewReader(os.Stdin),
		writer: bufio.NewWriter(os.Stdout),
	}
}

// Read implements Transport.Read
func (t *StdioTransport) Read() ([]byte, error) {
	line, err := t.reader.ReadBytes('\n')
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("stdin closed")
		}
		return nil, fmt.Errorf("failed to read from stdin: %w", err)
	}
	return line, nil
}

// Write implements Transport.Write
func (t *StdioTransport) Write(data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, err := t.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write to stdout: %w", err)
	}

	if err := t.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush stdout: %w", err)
	}

	return nil
}

// Close implements Transport.Close
func (t *StdioTransport) Close() error {
	// Stdio doesn't need explicit closing
	return nil
}

// TransportFactory creates transport instances
type TransportFactory struct{}

// NewTransportFactory creates a new transport factory
func NewTransportFactory() *TransportFactory {
	return &TransportFactory{}
}

// CreateStdioTransport creates a stdio transport
func (f *TransportFactory) CreateStdioTransport() Transport {
	return NewStdioTransport()
}