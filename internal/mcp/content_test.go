package mcp

import (
	"errors"
	"testing"
)

// Test TextContent implementation
func TestTextContent(t *testing.T) {
	text := "Hello, MCP World!"
	content := NewTextContent(text)

	// Test type
	if content.Type() != "text" {
		t.Errorf("Expected type 'text', got '%s'", content.Type())
	}

	// Test text retrieval
	if content.GetText() != text {
		t.Errorf("Expected text '%s', got '%s'", text, content.GetText())
	}

	// Test blob retrieval (should be text as bytes)
	expectedBlob := []byte(text)
	blob := content.GetBlob()
	if string(blob) != text {
		t.Errorf("Expected blob '%s', got '%s'", text, string(blob))
	}

	// Compare byte slices
	if len(blob) != len(expectedBlob) {
		t.Errorf("Expected blob length %d, got %d", len(expectedBlob), len(blob))
	}
	for i, b := range blob {
		if b != expectedBlob[i] {
			t.Errorf("Blob mismatch at index %d: expected %d, got %d", i, expectedBlob[i], b)
		}
	}
}

// Test TextContent with empty string
func TestTextContentEmpty(t *testing.T) {
	content := NewTextContent("")

	if content.Type() != "text" {
		t.Errorf("Expected type 'text', got '%s'", content.Type())
	}

	if content.GetText() != "" {
		t.Errorf("Expected empty text, got '%s'", content.GetText())
	}

	blob := content.GetBlob()
	if len(blob) != 0 {
		t.Errorf("Expected empty blob, got length %d", len(blob))
	}
}

// Test TextContent with special characters
func TestTextContentSpecialChars(t *testing.T) {
	text := "Hello 🌍\nMultiline\tWith\ttabs\r\nWindows line endings"
	content := NewTextContent(text)

	if content.GetText() != text {
		t.Errorf("Expected text with special chars preserved, got '%s'", content.GetText())
	}

	if string(content.GetBlob()) != text {
		t.Errorf("Expected blob with special chars preserved, got '%s'", string(content.GetBlob()))
	}
}

// Test BlobContent implementation
func TestBlobContent(t *testing.T) {
	data := []byte{0x48, 0x65, 0x6C, 0x6C, 0x6F, 0x00, 0xFF, 0x01} // "Hello" + binary data
	content := NewBlobContent(data)

	// Test type
	if content.Type() != "blob" {
		t.Errorf("Expected type 'blob', got '%s'", content.Type())
	}

	// Test blob retrieval
	blob := content.GetBlob()
	if len(blob) != len(data) {
		t.Errorf("Expected blob length %d, got %d", len(data), len(blob))
	}
	for i, b := range blob {
		if b != data[i] {
			t.Errorf("Blob mismatch at index %d: expected %d, got %d", i, data[i], b)
		}
	}

	// Test text retrieval (should be blob as string)
	expectedText := string(data)
	if content.GetText() != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, content.GetText())
	}
}

// Test BlobContent with empty data
func TestBlobContentEmpty(t *testing.T) {
	content := NewBlobContent([]byte{})

	if content.Type() != "blob" {
		t.Errorf("Expected type 'blob', got '%s'", content.Type())
	}

	if len(content.GetBlob()) != 0 {
		t.Errorf("Expected empty blob, got length %d", len(content.GetBlob()))
	}

	if content.GetText() != "" {
		t.Errorf("Expected empty text, got '%s'", content.GetText())
	}
}

// Test BlobContent with nil data
func TestBlobContentNil(t *testing.T) {
	content := NewBlobContent(nil)

	if content.Type() != "blob" {
		t.Errorf("Expected type 'blob', got '%s'", content.Type())
	}

	blob := content.GetBlob()
	if blob != nil {
		t.Errorf("Expected nil blob, got %v", blob)
	}

	if content.GetText() != "" {
		t.Errorf("Expected empty text for nil blob, got '%s'", content.GetText())
	}
}

// Test successful ToolResult
func TestToolResultSuccess(t *testing.T) {
	textContent := NewTextContent("Success message")
	blobContent := NewBlobContent([]byte("Binary data"))
	
	result := NewToolResult(textContent, blobContent)

	// Test error status
	if result.IsError() {
		t.Error("Expected successful result, but IsError() returned true")
	}

	// Test error retrieval
	if result.GetError() != nil {
		t.Errorf("Expected no error, got '%v'", result.GetError())
	}

	// Test content retrieval
	content := result.GetContent()
	if len(content) != 2 {
		t.Errorf("Expected 2 content items, got %d", len(content))
	}

	if content[0].GetText() != "Success message" {
		t.Errorf("Expected first content 'Success message', got '%s'", content[0].GetText())
	}

	if content[1].Type() != "blob" {
		t.Errorf("Expected second content type 'blob', got '%s'", content[1].Type())
	}
}

// Test empty successful ToolResult
func TestToolResultSuccessEmpty(t *testing.T) {
	result := NewToolResult()

	if result.IsError() {
		t.Error("Expected successful result, but IsError() returned true")
	}

	if result.GetError() != nil {
		t.Errorf("Expected no error, got '%v'", result.GetError())
	}

	content := result.GetContent()
	if len(content) != 0 {
		t.Errorf("Expected 0 content items, got %d", len(content))
	}
}

// Test error ToolResult
func TestToolResultError(t *testing.T) {
	testError := errors.New("tool execution failed")
	result := NewToolError(testError)

	// Test error status
	if !result.IsError() {
		t.Error("Expected error result, but IsError() returned false")
	}

	// Test error retrieval
	if result.GetError() != testError {
		t.Errorf("Expected error '%v', got '%v'", testError, result.GetError())
	}

	// Test content retrieval (should be empty for error)
	content := result.GetContent()
	if content != nil {
		t.Errorf("Expected nil content for error result, got %v", content)
	}
}

// Test error ToolResult with nil error
func TestToolResultErrorNil(t *testing.T) {
	result := NewToolError(nil)

	// With nil error, it should still be considered an error result
	// because it was created via NewToolError
	if !result.IsError() {
		t.Error("Expected error result even with nil error")
	}

	if result.GetError() != nil {
		t.Errorf("Expected nil error, got '%v'", result.GetError())
	}
}

// Test ResourceContent implementation
func TestResourceContent(t *testing.T) {
	textContent := NewTextContent("Resource text")
	blobContent := NewBlobContent([]byte("Resource binary"))
	mimeType := "application/json"

	resourceContent := NewResourceContent(mimeType, textContent, blobContent)

	// Test MIME type
	if resourceContent.GetMimeType() != mimeType {
		t.Errorf("Expected MIME type '%s', got '%s'", mimeType, resourceContent.GetMimeType())
	}

	// Test content retrieval
	content := resourceContent.GetContent()
	if len(content) != 2 {
		t.Errorf("Expected 2 content items, got %d", len(content))
	}

	if content[0].GetText() != "Resource text" {
		t.Errorf("Expected first content 'Resource text', got '%s'", content[0].GetText())
	}

	if content[1].Type() != "blob" {
		t.Errorf("Expected second content type 'blob', got '%s'", content[1].Type())
	}
}

// Test ResourceContent with empty content
func TestResourceContentEmpty(t *testing.T) {
	mimeType := "text/plain"
	resourceContent := NewResourceContent(mimeType)

	if resourceContent.GetMimeType() != mimeType {
		t.Errorf("Expected MIME type '%s', got '%s'", mimeType, resourceContent.GetMimeType())
	}

	content := resourceContent.GetContent()
	if len(content) != 0 {
		t.Errorf("Expected 0 content items, got %d", len(content))
	}
}

// Test ResourceContent with empty MIME type
func TestResourceContentEmptyMimeType(t *testing.T) {
	textContent := NewTextContent("Some content")
	resourceContent := NewResourceContent("", textContent)

	if resourceContent.GetMimeType() != "" {
		t.Errorf("Expected empty MIME type, got '%s'", resourceContent.GetMimeType())
	}

	content := resourceContent.GetContent()
	if len(content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(content))
	}
}

// Test interface compliance for content types
func TestContentInterfaceCompliance(t *testing.T) {
	// Ensure our concrete types implement the interfaces
	var _ Content = (*TextContent)(nil)
	var _ Content = (*BlobContent)(nil)
	var _ ToolResult = (*toolResult)(nil)
	var _ ResourceContent = (*resourceContent)(nil)
}

// Benchmark tests for performance-critical operations
func BenchmarkTextContentCreation(b *testing.B) {
	text := "This is a test string for benchmarking"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewTextContent(text)
	}
}

func BenchmarkBlobContentCreation(b *testing.B) {
	data := make([]byte, 1024) // 1KB of data
	for i := range data {
		data[i] = byte(i % 256)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewBlobContent(data)
	}
}

func BenchmarkToolResultCreation(b *testing.B) {
	content := NewTextContent("Benchmark content")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewToolResult(content)
	}
}