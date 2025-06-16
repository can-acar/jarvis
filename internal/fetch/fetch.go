package fetching

import (
	"jarvis/handlers"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterFetchTools registers all web fetching MCP tools
func RegisterFetchTools(s *server.MCPServer) {
	// fetch_web - General HTTP request
	fetchWeb := mcp.NewTool("fetch_web",
		mcp.WithDescription("Fetch represents a structured HTTP request for fetching resources"),
		mcp.WithString("url", mcp.Required(), mcp.Description("URL to fetch")),
		mcp.WithString("method", mcp.Description("HTTP method (default: GET)")),
		mcp.WithString("headers", mcp.Description("HTTP headers as JSON string")),
		mcp.WithString("body", mcp.Description("Request body")),
		mcp.WithNumber("timeout", mcp.Description("Request timeout in seconds (default: 30)")),
		mcp.WithBoolean("follow_redirects", mcp.Description("Follow HTTP redirects (default: true)")),
		mcp.WithNumber("max_redirects", mcp.Description("Maximum number of redirects to follow (default: 10)")),
	)
	s.AddTool(fetchWeb, handlers.HandleFetchWeb)

	// fetch_web_content - Web content fetching
	fetchWebContent := mcp.NewTool("fetch_web_content",
		mcp.WithDescription("Fetch web content with options for headers, method, and body"),
		mcp.WithString("url", mcp.Required(), mcp.Description("URL to fetch content from")),
		mcp.WithString("method", mcp.Description("HTTP method (default: GET)")),
		mcp.WithString("headers", mcp.Description("HTTP headers as JSON string")),
		mcp.WithString("body", mcp.Description("Request body")),
		mcp.WithString("user_agent", mcp.Description("Custom User-Agent string")),
		mcp.WithBoolean("include_headers", mcp.Description("Include response headers in output (default: false)")),
		mcp.WithString("encoding", mcp.Description("Expected content encoding (default: auto-detect)")),
	)
	s.AddTool(fetchWebContent, handlers.HandleFetchWebContent)

	// fetch_web_file - File download
	fetchWebFile := mcp.NewTool("fetch_web_file",
		mcp.WithDescription("Fetch a file from a URL and save it locally"),
		mcp.WithString("url", mcp.Required(), mcp.Description("URL of the file to download")),
		mcp.WithString("filepath", mcp.Required(), mcp.Description("Local path to save the file")),
		mcp.WithString("headers", mcp.Description("HTTP headers as JSON string")),
		mcp.WithBoolean("overwrite", mcp.Description("Overwrite existing file (default: false)")),
		mcp.WithBoolean("resume", mcp.Description("Resume partial downloads (default: false)")),
		mcp.WithBoolean("verify_checksum", mcp.Description("Verify file integrity if checksum available (default: false)")),
		mcp.WithString("expected_checksum", mcp.Description("Expected file checksum (SHA256)")),
	)
	s.AddTool(fetchWebFile, handlers.HandleFetchWebFile)

	// fetch_web_image - Image download with validation
	fetchWebImage := mcp.NewTool("fetch_web_image",
		mcp.WithDescription("Fetch an image from a URL and save it locally"),
		mcp.WithString("url", mcp.Required(), mcp.Description("URL of the image to download")),
		mcp.WithString("filepath", mcp.Required(), mcp.Description("Local path to save the image")),
		mcp.WithString("format", mcp.Description("Expected image format (jpg, png, gif, webp)")),
		mcp.WithBoolean("validate_image", mcp.Description("Validate that downloaded content is an image (default: true)")),
		mcp.WithNumber("max_size_mb", mcp.Description("Maximum file size in MB (default: 50)")),
		mcp.WithBoolean("convert_format", mcp.Description("Convert to specified format if different (default: false)")),
		mcp.WithString("quality", mcp.Description("Image quality for conversion (default: 85)")),
	)
	s.AddTool(fetchWebImage, handlers.HandleFetchWebImage)

	// fetch_web_json - JSON API fetching
	fetchWebJSON := mcp.NewTool("fetch_web_json",
		mcp.WithDescription("Fetch JSON data from a URL and parse it"),
		mcp.WithString("url", mcp.Required(), mcp.Description("URL to fetch JSON from")),
		mcp.WithString("headers", mcp.Description("HTTP headers as JSON string")),
		mcp.WithString("method", mcp.Description("HTTP method (default: GET)")),
		mcp.WithString("body", mcp.Description("Request body for POST/PUT requests")),
		mcp.WithBoolean("pretty_print", mcp.Description("Pretty print JSON response (default: true)")),
		mcp.WithString("json_path", mcp.Description("JSONPath expression to extract specific data")),
		mcp.WithBoolean("validate_schema", mcp.Description("Validate JSON against expected schema (default: false)")),
	)
	s.AddTool(fetchWebJSON, handlers.HandleFetchWebJSON)

	// fetch_web_batch - Batch URL fetching
	fetchWebBatch := mcp.NewTool("fetch_web_batch",
		mcp.WithDescription("Fetch multiple URLs concurrently"),
		mcp.WithString("urls", mcp.Required(), mcp.Description("JSON array of URL configurations")),
		mcp.WithNumber("max_concurrent", mcp.Description("Maximum concurrent requests (default: 5)")),
		mcp.WithNumber("delay_ms", mcp.Description("Delay between requests in milliseconds (default: 0)")),
		mcp.WithBoolean("fail_fast", mcp.Description("Stop on first error (default: false)")),
		mcp.WithBoolean("include_timing", mcp.Description("Include timing information (default: true)")),
	)
	s.AddTool(fetchWebBatch, handlers.HandleFetchWebBatch)

	// check_url_status - URL health check
	checkURLStatus := mcp.NewTool("check_url_status",
		mcp.WithDescription("Check the status and availability of one or more URLs"),
		mcp.WithString("urls", mcp.Required(), mcp.Description("Single URL or JSON array of URLs to check")),
		mcp.WithNumber("timeout", mcp.Description("Request timeout in seconds (default: 10)")),
		mcp.WithBoolean("follow_redirects", mcp.Description("Follow redirects (default: true)")),
		mcp.WithBoolean("check_ssl", mcp.Description("Check SSL certificate validity (default: true)")),
		mcp.WithBoolean("include_headers", mcp.Description("Include response headers (default: false)")),
	)
	s.AddTool(checkURLStatus, handlers.HandleCheckURLStatus)
}
