package prompt

import (
	"context"

	"github.com/yosida95/uritemplate/v3"
)

// Resource represents a known resource that the server can read.
type Resource struct {
	// Resource name
	Name string `json:"name"`

	// Resource URI
	URI string `json:"uri"`

	// Resource description (optional)
	Description string `json:"description,omitempty"`

	// MIME type (optional)
	MimeType string `json:"mimeType,omitempty"`

	// Resource size in bytes (optional)
	Size int64 `json:"size,omitempty"`

	// Annotations (optional)
	Annotated
}

// ResourceContents represents resource contents
type ResourceContents interface {
	isResourceContents()
}

// ReadResourceRequest describes a request to read a resource.
type ReadResourceRequest struct {
	Request
	Params struct {
		URI       string                 `json:"uri"`
		Arguments map[string]interface{} `json:"arguments,omitempty"`
	} `json:"params"`
}

// ReadResourceResult describes a result of reading a resource.
type ReadResourceResult struct {
	Result
	Contents []ResourceContents `json:"contents"`
}

// resourceHandler defines the function type for handling resource reading
type resourceHandler func(ctx context.Context, req *ReadResourceRequest) (ResourceContents, error)

// resourceTemplateHandler defines the function type for handling resource template reading.
type resourceTemplateHandler func(ctx context.Context, req *ReadResourceRequest) ([]ResourceContents, error)

// registeredResource combines a Resource with its handler function
type registeredResource struct {
	Resource *Resource
	Handler  resourceHandler
}

// URITemplate represents a URI template.
type URITemplate struct {
	*uritemplate.Template
}

// ResourceTemplate describes a resource template
type ResourceTemplate struct {
	// Template name
	Name string `json:"name"`

	// URI template
	URITemplate *URITemplate `json:"uriTemplate"`

	// Resource description (optional)
	Description string `json:"description,omitempty"`

	// MIME type (optional)
	MimeType string `json:"mimeType,omitempty"`

	// Embed Annotated struct
	Annotated
}

// registerResourceTemplate 用于将一个资源模板 ResourceTemplate 与其对应的处理函数 resourceTemplateHandler 关联起来。
type registerResourceTemplate struct {
	resourceTemplate *ResourceTemplate
	Handler          resourceTemplateHandler
}

// ListResourcesResult describes a result of listing resources.
type ListResourcesResult struct {
	PaginatedResult
	Resources []Resource `json:"resources"`
}
