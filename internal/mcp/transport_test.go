package mcp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
)

type TestableStdioTransport struct {
	reader    io.Reader
	writer    io.Writer
	mu        sync.Mutex
	bufReader *bufio.Reader
	bufWriter *bufio.Writer
}

func NewTestableStdioTransport(reader io.Reader, writer io.Writer) *TestableStdioTransport {
	return &TestableStdioTransport{
		reader:    reader,
		writer:    writer,
		bufReader: bufio.NewReader(reader),
		bufWriter: bufio.NewWriter(writer),
	}
}

func (t *TestableStdioTransport) Read() ([]byte, error) {
	line, err := t.bufReader.ReadBytes('\n')
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("reader closed")
		}
		return nil, fmt.Errorf("failed to read: %w", err)
	}
	return line, nil
}

func (t *TestableStdioTransport) Write(data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, err := t.bufWriter.Write(data); err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}

	if err := t.bufWriter.Flush(); err != nil {
		return fmt.Errorf("failed to flush: %w", err)
	}

	return nil
}

func (t *TestableStdioTransport) Close() error {
	return nil
}

func TestStdioTransportReadWrite(t *testing.T) {
	input := "test message\n"
	reader := strings.NewReader(input)
	var writer bytes.Buffer

	transport := NewTestableStdioTransport(reader, &writer)

	data, err := transport.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if string(data) != input {
		t.Errorf("Expected '%s', got '%s'", input, string(data))
	}

	output := []byte("response message\n")
	if err := transport.Write(output); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if writer.String() != string(output) {
		t.Errorf("Expected written '%s', got '%s'", string(output), writer.String())
	}
}

func TestStdioTransportMultipleReads(t *testing.T) {
	input := "line1\nline2\nline3\n"
	reader := strings.NewReader(input)
	var writer bytes.Buffer

	transport := NewTestableStdioTransport(reader, &writer)

	// Read first line
	data1, err := transport.Read()
	if err != nil {
		t.Fatalf("First read failed: %v", err)
	}
	if string(data1) != "line1\n" {
		t.Errorf("Expected 'line1\\n', got '%s'", string(data1))
	}

	// Read second line
	data2, err := transport.Read()
	if err != nil {
		t.Fatalf("Second read failed: %v", err)
	}
	if string(data2) != "line2\n" {
		t.Errorf("Expected 'line2\\n', got '%s'", string(data2))
	}

	// Read third line
	data3, err := transport.Read()
	if err != nil {
		t.Fatalf("Third read failed: %v", err)
	}
	if string(data3) != "line3\n" {
		t.Errorf("Expected 'line3\\n', got '%s'", string(data3))
	}
}

func TestStdioTransportReadEOF(t *testing.T) {
	reader := strings.NewReader("") // Empty reader
	var writer bytes.Buffer

	transport := NewTestableStdioTransport(reader, &writer)

	// Reading from empty reader should return EOF error
	_, err := transport.Read()
	if err == nil {
		t.Error("Expected EOF error, got nil")
	}

	if !strings.Contains(err.Error(), "reader closed") {
		t.Errorf("Expected 'reader closed' error, got: %v", err)
	}
}

func TestStdioTransportWriteError(t *testing.T) {
	reader := strings.NewReader("test\n")

	// Create a writer that always returns an error
	errorWriter := &errorWriter{}

	transport := NewTestableStdioTransport(reader, errorWriter)

	// Writing should return an error
	err := transport.Write([]byte("test data"))
	if err == nil {
		t.Error("Expected write error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to write") && !strings.Contains(err.Error(), "failed to flush") {
		t.Errorf("Expected 'failed to write' or 'failed to flush' error, got: %v", err)
	}
}

func TestStdioTransportConcurrentWrites(t *testing.T) {
	reader := strings.NewReader("test\n")
	var writer bytes.Buffer

	transport := NewTestableStdioTransport(reader, &writer)

	numWrites := 10
	var wg sync.WaitGroup
	wg.Add(numWrites)

	// Start concurrent writes
	for i := 0; i < numWrites; i++ {
		go func(id int) {
			defer wg.Done()
			message := fmt.Sprintf("message-%d\n", id)
			if err := transport.Write([]byte(message)); err != nil {
				t.Errorf("Write %d failed: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all messages were written (order not guaranteed due to concurrency)
	output := writer.String()
	for i := 0; i < numWrites; i++ {
		expected := fmt.Sprintf("message-%d\n", i)
		if !strings.Contains(output, expected) {
			t.Errorf("Output missing expected message: %s", expected)
		}
	}
}

func TestStdioTransportClose(t *testing.T) {
	reader := strings.NewReader("test\n")
	var writer bytes.Buffer

	transport := NewTestableStdioTransport(reader, &writer)

	if err := transport.Close(); err != nil {
		t.Errorf("Close returned unexpected error: %v", err)
	}
}

func TestStdioTransportEmptyWrite(t *testing.T) {
	reader := strings.NewReader("test\n")
	var writer bytes.Buffer

	transport := NewTestableStdioTransport(reader, &writer)

	// Writing empty data should work
	if err := transport.Write([]byte{}); err != nil {
		t.Errorf("Empty write failed: %v", err)
	}

	if writer.Len() != 0 {
		t.Errorf("Expected empty output, got %d bytes", writer.Len())
	}
}

func TestStdioTransportReadWithoutNewline(t *testing.T) {
	reader := strings.NewReader("no newline")
	var writer bytes.Buffer

	transport := NewTestableStdioTransport(reader, &writer)

	// Reading without newline should return EOF error
	_, err := transport.Read()
	if err == nil {
		t.Error("Expected EOF error for input without newline")
	}
}

func TestStdioTransportLargeData(t *testing.T) {
	// Create large input (10KB)
	largeData := strings.Repeat("a", 10*1024) + "\n"
	reader := strings.NewReader(largeData)
	var writer bytes.Buffer

	transport := NewTestableStdioTransport(reader, &writer)

	// Read large data
	data, err := transport.Read()
	if err != nil {
		t.Fatalf("Large data read failed: %v", err)
	}

	if string(data) != largeData {
		t.Errorf("Large data mismatch, expected %d bytes, got %d bytes", len(largeData), len(data))
	}

	// Write large data
	largeOutput := []byte(strings.Repeat("b", 10*1024) + "\n")
	if err := transport.Write(largeOutput); err != nil {
		t.Fatalf("Large data write failed: %v", err)
	}

	if writer.String() != string(largeOutput) {
		t.Errorf("Large data write mismatch, expected %d bytes, got %d bytes", len(largeOutput), writer.Len())
	}
}

type errorWriter struct{}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("mock write error")
}

// Benchmark tests
func BenchmarkStdioTransportWrite(b *testing.B) {
	reader := strings.NewReader("test\n")
	var writer bytes.Buffer
	transport := NewTestableStdioTransport(reader, &writer)

	data := []byte("benchmark message\n")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := transport.Write(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdioTransportRead(b *testing.B) {
	// Create a large input for benchmarking
	var inputBuilder strings.Builder
	for i := 0; i < b.N; i++ {
		inputBuilder.WriteString("benchmark line\n")
	}

	reader := strings.NewReader(inputBuilder.String())
	var writer bytes.Buffer
	transport := NewTestableStdioTransport(reader, &writer)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := transport.Read()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestRealStdioTransportCreation(t *testing.T) {
	transport := NewStdioTransport()
	if transport == nil {
		t.Fatal("Real stdio transport creation failed")
	}

	// Test close (should not error)
	if err := transport.Close(); err != nil {
		t.Errorf("Real stdio transport close failed: %v", err)
	}
}
