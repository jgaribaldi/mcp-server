package mcp

type TextContent struct {
	text string
}

func NewTextContent(text string) Content {
	return &TextContent{text: text}
}

func (c *TextContent) Type() string {
	return "text"
}

func (c *TextContent) GetText() string {
	return c.text
}

func (c *TextContent) GetBlob() []byte {
	return []byte(c.text)
}

type BlobContent struct {
	data []byte
}

func NewBlobContent(data []byte) Content {
	return &BlobContent{data: data}
}

func (c *BlobContent) Type() string {
	return "blob"
}

func (c *BlobContent) GetText() string {
	return string(c.data)
}

func (c *BlobContent) GetBlob() []byte {
	return c.data
}

type toolResult struct {
	content []Content
	error   error
	isError bool
}

func NewToolResult(content ...Content) ToolResult {
	return &toolResult{content: content, isError: false}
}

func NewToolError(err error) ToolResult {
	return &toolResult{error: err, isError: true}
}

func (r *toolResult) IsError() bool {
	return r.isError
}

func (r *toolResult) GetContent() []Content {
	return r.content
}

func (r *toolResult) GetError() error {
	return r.error
}

type resourceContent struct {
	content  []Content
	mimeType string
}

func NewResourceContent(mimeType string, content ...Content) ResourceContent {
	return &resourceContent{
		content:  content,
		mimeType: mimeType,
	}
}

func (r *resourceContent) GetContent() []Content {
	return r.content
}

func (r *resourceContent) GetMimeType() string {
	return r.mimeType
}
