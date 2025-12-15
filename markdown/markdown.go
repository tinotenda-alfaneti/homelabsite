package markdown

import (
	"html/template"
	"strings"
)

func Render(content string) template.HTML {
	return template.HTML(renderMarkdown(content))
}

func renderMarkdown(content string) string {
	// Simple markdown rendering - headers, bold, lists, code blocks
	lines := strings.Split(content, "\n")
	var html strings.Builder
	inCodeBlock := false
	inList := false

	for i, line := range lines {
		// Code blocks
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				html.WriteString("</code></pre>")
				inCodeBlock = false
			} else {
				html.WriteString("<pre><code>")
				inCodeBlock = true
			}
			continue
		}

		if inCodeBlock {
			html.WriteString(template.HTMLEscapeString(line) + "\n")
			continue
		}

		// Headers
		if strings.HasPrefix(line, "## ") {
			if inList {
				html.WriteString("</ul>")
				inList = false
			}
			html.WriteString("<h2>" + template.HTMLEscapeString(strings.TrimPrefix(line, "## ")) + "</h2>")
			continue
		}
		if strings.HasPrefix(line, "### ") {
			if inList {
				html.WriteString("</ul>")
				inList = false
			}
			html.WriteString("<h3>" + template.HTMLEscapeString(strings.TrimPrefix(line, "### ")) + "</h3>")
			continue
		}

		// Lists
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			if !inList {
				html.WriteString("<ul>")
				inList = true
			}
			text := strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* ")
			text = processBold(text)
			html.WriteString("<li>" + text + "</li>")
			continue
		}

		// Numbered lists
		if len(line) > 2 && line[0] >= '0' && line[0] <= '9' && line[1] == '.' {
			if !inList {
				html.WriteString("<ol>")
				inList = true
			}
			parts := strings.SplitN(line, ". ", 2)
			if len(parts) == 2 {
				text := processBold(parts[1])
				html.WriteString("<li>" + text + "</li>")
			}
			continue
		}

		// End list if we hit a non-list line
		if inList && line != "" && !strings.HasPrefix(line, "- ") && !strings.HasPrefix(line, "* ") {
			html.WriteString("</ul>")
			inList = false
		}

		// Empty lines
		if strings.TrimSpace(line) == "" {
			if i > 0 && i < len(lines)-1 {
				html.WriteString("<br>")
			}
			continue
		}

		// Regular paragraphs
		if !inList {
			text := processBold(line)
			html.WriteString("<p>" + text + "</p>")
		}
	}

	if inList {
		html.WriteString("</ul>")
	}
	if inCodeBlock {
		html.WriteString("</code></pre>")
	}

	return html.String()
}

func processBold(text string) string {
	// First escape the text
	escaped := template.HTMLEscapeString(text)
	// Then handle **bold** text
	for strings.Contains(escaped, "**") {
		first := strings.Index(escaped, "**")
		if first == -1 {
			break
		}
		second := strings.Index(escaped[first+2:], "**")
		if second == -1 {
			break
		}
		second += first + 2
		escaped = escaped[:first] + "<strong>" + escaped[first+2:second] + "</strong>" + escaped[second+2:]
	}
	return escaped
}
