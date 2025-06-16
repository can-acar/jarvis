package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"jarvis/internal/common"
	"jarvis/internal/types"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// ==================== HANDLER IMPLEMENTATIONS ====================

func HandleFetchWeb(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := req.RequireString("url")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid URL parameter: %v", err)), nil
	}

	if err := common.ValidateURL(url); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid URL: %v", err)), nil
	}

	method := mcp.ParseString(req, "method", "GET")
	timeout := time.Duration(mcp.ParseFloat64(req, "timeout", 30)) * time.Second
	followRedirects := mcp.ParseBoolean(req, "follow_redirects", true)
	maxRedirects := int(mcp.ParseFloat64(req, "max_redirects", 10))

	// Create HTTP client
	client := &http.Client{
		Timeout: timeout,
	}

	if !followRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else if maxRedirects > 0 {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			return nil
		}
	}

	// Create request
	var bodyReader io.Reader
	if body := mcp.ParseString(req, "body", ""); body != "" {
		bodyReader = strings.NewReader(body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create request: %v", err)), nil
	}

	// Set headers
	httpReq.Header.Set("User-Agent", common.BuildUserAgent("Jarvis-MCP", "1.0.0"))

	if headersStr := mcp.ParseString(req, "headers", ""); headersStr != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(headersStr), &headers); err == nil {
			for key, value := range headers {
				httpReq.Header.Set(key, value)
			}
		}
	}

	// Execute request
	start := time.Now()
	resp, err := client.Do(httpReq)
	duration := time.Since(start)

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Request failed: %v", err)), nil
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to read response: %v", err)), nil
	}

	// Format result
	result := fmt.Sprintf("Status: %s\n", resp.Status)
	result += fmt.Sprintf("Duration: %s\n", common.FormatDuration(duration))
	result += fmt.Sprintf("Content-Length: %s\n", common.FormatBytes(int64(len(body))))
	result += fmt.Sprintf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
	result += fmt.Sprintf("\nHeaders:\n")
	for key, values := range resp.Header {
		result += fmt.Sprintf("  %s: %s\n", key, strings.Join(values, ", "))
	}
	result += fmt.Sprintf("\nBody:\n%s", string(body))

	return mcp.NewToolResultText(result), nil
}

func HandleFetchWebContent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := req.RequireString("url")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid URL parameter: %v", err)), nil
	}

	if err := common.ValidateURL(url); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid URL: %v", err)), nil
	}

	method := mcp.ParseString(req, "method", "GET")
	userAgent := mcp.ParseString(req, "user_agent", common.BuildUserAgent("Jarvis-MCP", "1.0.0"))
	includeHeaders := mcp.ParseBoolean(req, "include_headers", false)

	client := &http.Client{Timeout: 30 * time.Second}

	var bodyReader io.Reader
	if body := mcp.ParseString(req, "body", ""); body != "" {
		bodyReader = strings.NewReader(body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create request: %v", err)), nil
	}

	httpReq.Header.Set("User-Agent", userAgent)

	// Set additional headers
	if headersStr := mcp.ParseString(req, "headers", ""); headersStr != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(headersStr), &headers); err == nil {
			for key, value := range headers {
				httpReq.Header.Set(key, value)
			}
		}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Request failed: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return mcp.NewToolResultError(fmt.Sprintf("HTTP Error %d: %s", resp.StatusCode, resp.Status)), nil
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to read content: %v", err)), nil
	}

	result := string(content)
	if includeHeaders {
		headerInfo := fmt.Sprintf("Status: %s\n", resp.Status)
		for key, values := range resp.Header {
			headerInfo += fmt.Sprintf("%s: %s\n", key, strings.Join(values, ", "))
		}
		result = headerInfo + "\n" + result
	}

	return mcp.NewToolResultText(result), nil
}

func HandleFetchWebFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := req.RequireString("url")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid URL parameter: %v", err)), nil
	}

	filePath, err := req.RequireString("filepath")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid filepath parameter: %v", err)), nil
	}

	if !common.IsPathAllowed(filePath) {
		return mcp.NewToolResultError("Access to this path is not allowed"), nil
	}

	overwrite := mcp.ParseBoolean(req, "overwrite", false)
	resume := mcp.ParseBoolean(req, "resume", false)
	verifyChecksum := mcp.ParseBoolean(req, "verify_checksum", false)
	expectedChecksum := mcp.ParseString(req, "expected_checksum", "")

	// Check if file exists
	existingSize := int64(0)
	if stat, err := os.Stat(filePath); err == nil {
		if !overwrite && !resume {
			return mcp.NewToolResultError("File already exists and overwrite is false"), nil
		}
		if resume {
			existingSize = stat.Size()
		}
	}

	client := &http.Client{Timeout: 10 * time.Minute}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create request: %v", err)), nil
	}

	// Set headers
	httpReq.Header.Set("User-Agent", common.BuildUserAgent("Jarvis-MCP", "1.0.0"))

	// Handle resume
	if resume && existingSize > 0 {
		httpReq.Header.Set("Range", fmt.Sprintf("bytes=%d-", existingSize))
	}

	if headersStr := mcp.ParseString(req, "headers", ""); headersStr != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(headersStr), &headers); err == nil {
			for key, value := range headers {
				httpReq.Header.Set(key, value)
			}
		}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Download failed: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return mcp.NewToolResultError(fmt.Sprintf("HTTP Error %d: %s", resp.StatusCode, resp.Status)), nil
	}

	// Create directory
	if err := common.EnsureDir(filepath.Dir(filePath)); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create directory: %v", err)), nil
	}

	// Open file for writing
	var file *os.File
	if resume && existingSize > 0 && resp.StatusCode == 206 {
		file, err = os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	} else {
		file, err = os.Create(filePath)
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create file: %v", err)), nil
	}
	defer file.Close()

	// Download with progress tracking
	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to save file: %v", err)), nil
	}

	totalSize := existingSize + written
	if resume && existingSize > 0 {
		totalSize = existingSize + written
	} else {
		totalSize = written
	}

	// Verify checksum if requested
	if verifyChecksum && expectedChecksum != "" {
		actualChecksum, err := common.CalculateFileChecksum(filePath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to calculate checksum: %v", err)), nil
		}
		if actualChecksum != expectedChecksum {
			return mcp.NewToolResultError(fmt.Sprintf("Checksum mismatch. Expected: %s, Got: %s", expectedChecksum, actualChecksum)), nil
		}
	}

	return mcp.NewToolResultText(fmt.Sprintf("File downloaded successfully: %s (%s)", filePath, common.FormatBytes(totalSize))), nil
}

func HandleFetchWebImage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := req.RequireString("url")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid URL parameter: %v", err)), nil
	}

	filePath, err := req.RequireString("filepath")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid filepath parameter: %v", err)), nil
	}

	if !common.IsPathAllowed(filePath) {
		return mcp.NewToolResultError("Access to this path is not allowed"), nil
	}

	validateImage := mcp.ParseBoolean(req, "validate_image", true)
	expectedFormat := mcp.ParseString(req, "format", "")
	maxSizeMB := mcp.ParseFloat64(req, "max_size_mb", 50)
	convertFormat := mcp.ParseBoolean(req, "convert_format", false)

	client := &http.Client{Timeout: 5 * time.Minute}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create request: %v", err)), nil
	}

	httpReq.Header.Set("User-Agent", common.BuildUserAgent("Jarvis-MCP", "1.0.0"))
	httpReq.Header.Set("Accept", "image/*")

	resp, err := client.Do(httpReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Download failed: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return mcp.NewToolResultError(fmt.Sprintf("HTTP Error %d: %s", resp.StatusCode, resp.Status)), nil
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if validateImage && common.GetContentTypeCategory(contentType) != "image" {
		return mcp.NewToolResultError(fmt.Sprintf("Content is not an image: %s", contentType)), nil
	}

	// Check file size
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		if size, err := common.ParseInt64(contentLength); err == nil {
			sizeMB := float64(size) / (1024 * 1024)
			if sizeMB > maxSizeMB {
				return mcp.NewToolResultError(fmt.Sprintf("File too large: %.1fMB (max: %.1fMB)", sizeMB, maxSizeMB)), nil
			}
		}
	}

	// Validate format if specified
	if expectedFormat != "" {
		expectedMimeType := "image/" + expectedFormat
		if !strings.Contains(contentType, expectedMimeType) && !convertFormat {
			return mcp.NewToolResultError(fmt.Sprintf("Image format mismatch. Expected: %s, Got: %s", expectedMimeType, contentType)), nil
		}
	}

	// Create directory
	if err := common.EnsureDir(filepath.Dir(filePath)); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create directory: %v", err)), nil
	}

	// Download file
	file, err := os.Create(filePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create file: %v", err)), nil
	}
	defer file.Close()

	size, err := io.Copy(file, resp.Body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to save image: %v", err)), nil
	}

	result := fmt.Sprintf("Image downloaded successfully: %s (%s, %s)", filePath, common.FormatBytes(size), contentType)

	// Convert format if requested
	if convertFormat && expectedFormat != "" && !strings.Contains(contentType, "image/"+expectedFormat) {
		convertedPath, err := common.ConvertImageFormat(filePath, expectedFormat)
		if err != nil {
			result += fmt.Sprintf("\nWarning: Format conversion failed: %v", err)
		} else {
			result += fmt.Sprintf("\nConverted to: %s", convertedPath)
		}
	}

	return mcp.NewToolResultText(result), nil
}

func HandleFetchWebJSON(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := req.RequireString("url")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid URL parameter: %v", err)), nil
	}

	if err := common.ValidateURL(url); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid URL: %v", err)), nil
	}

	method := mcp.ParseString(req, "method", "GET")
	prettyPrint := mcp.ParseBoolean(req, "pretty_print", true)
	jsonPath := mcp.ParseString(req, "json_path", "")

	client := &http.Client{Timeout: 30 * time.Second}

	var bodyReader io.Reader
	if body := mcp.ParseString(req, "body", ""); body != "" {
		bodyReader = strings.NewReader(body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create request: %v", err)), nil
	}

	httpReq.Header.Set("User-Agent", common.BuildUserAgent("Jarvis-MCP", "1.0.0"))
	httpReq.Header.Set("Accept", "application/json")

	if method == "POST" || method == "PUT" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set additional headers
	if headersStr := mcp.ParseString(req, "headers", ""); headersStr != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(headersStr), &headers); err == nil {
			for key, value := range headers {
				httpReq.Header.Set(key, value)
			}
		}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Request failed: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return mcp.NewToolResultError(fmt.Sprintf("HTTP Error %d: %s", resp.StatusCode, resp.Status)), nil
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return mcp.NewToolResultError(fmt.Sprintf("Response is not JSON: %s", contentType)), nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to read response: %v", err)), nil
	}

	// Parse JSON
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid JSON response: %v", err)), nil
	}

	// Apply JSONPath if specified
	if jsonPath != "" {
		extractedData, err := common.ApplyJSONPath(jsonData, jsonPath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("JSONPath error: %v", err)), nil
		}
		jsonData = extractedData
	}

	// Format output
	if prettyPrint {
		prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
		if err != nil {
			return mcp.NewToolResultText(string(body)), nil // Fallback to raw
		}
		return mcp.NewToolResultText(string(prettyJSON)), nil
	}

	return mcp.NewToolResultText(string(body)), nil
}

func HandleFetchWebBatch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	urlsStr, err := req.RequireString("urls")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid urls parameter: %v", err)), nil
	}

	var urlConfigs []types.HTTPRequestConfig
	if err := json.Unmarshal([]byte(urlsStr), &urlConfigs); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse URLs: %v", err)), nil
	}

	maxConcurrent := int(mcp.ParseFloat64(req, "max_concurrent", 5))
	delayMs := int(mcp.ParseFloat64(req, "delay_ms", 0))
	failFast := mcp.ParseBoolean(req, "fail_fast", false)
	includeTiming := mcp.ParseBoolean(req, "include_timing", true)

	results, err := common.FetchURLsBatch(ctx, urlConfigs, maxConcurrent, delayMs, failFast, includeTiming)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Batch fetch failed: %v", err)), nil
	}

	// Format results
	output, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(output)), nil
}

func HandleCheckURLStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	urlsStr, err := req.RequireString("urls")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid urls parameter: %v", err)), nil
	}

	timeout := time.Duration(mcp.ParseFloat64(req, "timeout", 10)) * time.Second
	followRedirects := mcp.ParseBoolean(req, "follow_redirects", true)
	checkSSL := mcp.ParseBoolean(req, "check_ssl", true)
	includeHeaders := mcp.ParseBoolean(req, "include_headers", false)

	// Parse URLs (can be single URL or array)
	var urls []string
	if strings.HasPrefix(strings.TrimSpace(urlsStr), "[") {
		if err := json.Unmarshal([]byte(urlsStr), &urls); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to parse URLs array: %v", err)), nil
		}
	} else {
		urls = []string{urlsStr}
	}

	results, err := common.CheckURLsStatus(ctx, urls, timeout, followRedirects, checkSSL, includeHeaders)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("URL status check failed: %v", err)), nil
	}

	// Format results
	output, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(output)), nil
}
