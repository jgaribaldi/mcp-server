package mcp

type TextContent struct {
	Text string
}


func (c *TextContent) Type() string {
	return "text"
}

func (c *TextContent) GetText() string {
	return c.Text
}

func (c *TextContent) GetBlob() []byte {
	return []byte(c.Text)
}

type BlobContent struct {
	Data []byte
}


func (c *BlobContent) Type() string {
	return "blob"
}

func (c *BlobContent) GetText() string {
	return string(c.Data)
}

func (c *BlobContent) GetBlob() []byte {
	return c.Data
}

type ToolResultImpl struct {
	Content []Content
	Error   error
	IsErrorFlag bool
}



func (r *ToolResultImpl) IsError() bool {
	return r.IsErrorFlag
}

func (r *ToolResultImpl) GetContent() []Content {
	return r.Content
}

func (r *ToolResultImpl) GetError() error {
	return r.Error
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
