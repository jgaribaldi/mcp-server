package mcp

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
)

type StdioTransport struct {
	reader *bufio.Reader
	writer *bufio.Writer
	mu     sync.Mutex
}

func NewStdioTransport() Transport {
	return &StdioTransport{
		reader: bufio.NewReader(os.Stdin),
		writer: bufio.NewWriter(os.Stdout),
	}
}

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

func (t *StdioTransport) Close() error {
	// Stdio doesn't need explicit closing
	return nil
}

type TransportFactory struct{}

func NewTransportFactory() *TransportFactory {
	return &TransportFactory{}
}

func (f *TransportFactory) CreateStdioTransport() Transport {
	return NewStdioTransport()
}
