package mcp

// TextContent represents text content
type TextContent struct {
	text string
}

// NewTextContent creates new text content
func NewTextContent(text string) Content {
	return &TextContent{text: text}
}

// Type implements Content.Type
func (c *TextContent) Type() string {
	return "text"
}

// GetText implements Content.GetText
func (c *TextContent) GetText() string {
	return c.text
}

// GetBlob implements Content.GetBlob
func (c *TextContent) GetBlob() []byte {
	return []byte(c.text)
}

// BlobContent represents binary content
type BlobContent struct {
	data []byte
}

// NewBlobContent creates new blob content
func NewBlobContent(data []byte) Content {
	return &BlobContent{data: data}
}

// Type implements Content.Type
func (c *BlobContent) Type() string {
	return "blob"
}

// GetText implements Content.GetText
func (c *BlobContent) GetText() string {
	return string(c.data)
}

// GetBlob implements Content.GetBlob
func (c *BlobContent) GetBlob() []byte {
	return c.data
}

// toolResult implementation
type toolResult struct {
	content []Content
	error   error
	isError bool // Track if this was created as an error result
}

// NewToolResult creates a successful tool result
func NewToolResult(content ...Content) ToolResult {
	return &toolResult{content: content, isError: false}
}

// NewToolError creates an error tool result
func NewToolError(err error) ToolResult {
	return &toolResult{error: err, isError: true}
}

// IsError implements ToolResult.IsError
func (r *toolResult) IsError() bool {
	return r.isError
}

// GetContent implements ToolResult.GetContent
func (r *toolResult) GetContent() []Content {
	return r.content
}

// GetError implements ToolResult.GetError
func (r *toolResult) GetError() error {
	return r.error
}

// resourceContent implementation
type resourceContent struct {
	content  []Content
	mimeType string
}

// NewResourceContent creates new resource content
func NewResourceContent(mimeType string, content ...Content) ResourceContent {
	return &resourceContent{
		content:  content,
		mimeType: mimeType,
	}
}

// GetContent implements ResourceContent.GetContent
func (r *resourceContent) GetContent() []Content {
	return r.content
}

// GetMimeType implements ResourceContent.GetMimeType
func (r *resourceContent) GetMimeType() string {
	return r.mimeType
}