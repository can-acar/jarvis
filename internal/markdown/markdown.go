package markdown

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
)

// Renderer handles markdown rendering for terminal output
type Renderer struct {
	renderer *glamour.TermRenderer
}

// NewRenderer creates a new markdown renderer
func NewRenderer() (*Renderer, error) {
	// Create a custom style for better terminal output
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create markdown renderer: %v", err)
	}

	return &Renderer{
		renderer: r,
	}, nil
}

// Render converts markdown text to formatted terminal output
func (r *Renderer) Render(markdown string) (string, error) {
	if r.renderer == nil {
		// Fallback to plain text if renderer failed to initialize
		return markdown, nil
	}

	// Clean up the markdown text
	cleaned := r.cleanMarkdown(markdown)

	rendered, err := r.renderer.Render(cleaned)
	if err != nil {
		// Fallback to original text if rendering fails
		return markdown, nil
	}

	return strings.TrimSpace(rendered), nil
}

// RenderSafe renders markdown with fallback to plain text
func (r *Renderer) RenderSafe(text string) string {
	rendered, err := r.Render(text)
	if err != nil {
		return text
	}
	return rendered
}

// cleanMarkdown preprocesses markdown text for better rendering
func (r *Renderer) cleanMarkdown(text string) string {
	// Remove excessive whitespace
	text = strings.TrimSpace(text)

	// Ensure proper line endings for code blocks
	text = strings.ReplaceAll(text, "```\n\n", "```\n")
	text = strings.ReplaceAll(text, "\n\n```", "\n```")

	return text
}

// IsMarkdown checks if text contains markdown syntax
func IsMarkdown(text string) bool {
	// Simple heuristics to detect markdown
	markdownIndicators := []string{
		"**", "*", "__", "_", // Bold/italic
		"```", "`", // Code blocks/inline code
		"#", "##", "###", // Headers
		"- ", "* ", "+ ", // Lists
		"1. ", "2. ", // Numbered lists
		"[", "](", // Links
		"---", "***", // Horizontal rules
	}

	for _, indicator := range markdownIndicators {
		if strings.Contains(text, indicator) {
			return true
		}
	}

	return false
}

// FormatBotResponse formats a bot response with emojis and markdown
func (r *Renderer) FormatBotResponse(text string) string {
	// Add bot emoji prefix if not already present
	if !strings.HasPrefix(text, "🤖") {
		text = "🤖 " + text
	}

	// Render markdown if detected
	if IsMarkdown(text) {
		return r.RenderSafe(text)
	}

	return text
}

// FormatToolsUsed formats the tools used section
func (r *Renderer) FormatToolsUsed(tools []string) string {
	if len(tools) == 0 {
		return ""
	}

	toolsText := fmt.Sprintf("🔧 **Tools used**: %s", strings.Join(tools, ", "))
	return r.RenderSafe(toolsText)
}

// FormatError formats error messages
func (r *Renderer) FormatError(message string) string {
	errorText := fmt.Sprintf("❌ **Error**: %s", message)
	return r.RenderSafe(errorText)
}

// FormatSuccess formats success messages
func (r *Renderer) FormatSuccess(message string) string {
	successText := fmt.Sprintf("✅ **Success**: %s", message)
	return r.RenderSafe(successText)
}

// FormatConfirmation formats confirmation prompts
func (r *Renderer) FormatConfirmation(message string) string {
	confirmText := fmt.Sprintf("⚠️  **Confirmation Required**\n\n%s\n\n**Continue? (y/N):**", message)
	return r.RenderSafe(confirmText)
}
