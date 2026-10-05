package api

import (
	"encoding/xml"
	"net/http/httptest"
	"testing"
)

func TestWebhookEmptyTwiMLIsValidResponse(t *testing.T) {
	response := httptest.NewRecorder()
	(&WebhookHandler{}).respondTwiML(response, "")
	var document struct {
		XMLName  xml.Name `xml:"Response"`
		Messages []string `xml:"Message"`
	}
	if err := xml.Unmarshal(response.Body.Bytes(), &document); err != nil {
		t.Fatalf("Twilio must receive a complete XML response: %v", err)
	}
	if response.Code != 200 || len(document.Messages) != 0 {
		t.Fatal("An acknowledged incoming SMS must not generate a reply")
	}
}

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no special characters",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "ampersand",
			input:    "Tom & Jerry",
			expected: "Tom &amp; Jerry",
		},
		{
			name:     "less than",
			input:    "5 < 10",
			expected: "5 &lt; 10",
		},
		{
			name:     "greater than",
			input:    "10 > 5",
			expected: "10 &gt; 5",
		},
		{
			name:     "single quote",
			input:    "It's great",
			expected: "It&apos;s great",
		},
		{
			name:     "double quote",
			input:    `Say "hello"`,
			expected: "Say &quot;hello&quot;",
		},
		{
			name:     "multiple special characters",
			input:    `<script>alert("XSS & evil")</script>`,
			expected: "&lt;script&gt;alert(&quot;XSS &amp; evil&quot;)&lt;/script&gt;",
		},
		{
			name:     "XML-like content",
			input:    "<Say>Hello <World></Say>",
			expected: "&lt;Say&gt;Hello &lt;World&gt;&lt;/Say&gt;",
		},
		{
			name:     "all special chars in order",
			input:    `& < > ' "`,
			expected: "&amp; &lt; &gt; &apos; &quot;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeXML(tt.input)
			if got != tt.expected {
				t.Errorf("escapeXML(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEscapeXML_DoesNotDoubleEscape(t *testing.T) {
	// If we escape already-escaped content, it should still escape
	input := "&amp;"
	expected := "&amp;amp;"
	got := escapeXML(input)
	if got != expected {
		t.Errorf("escapeXML(%q) = %q, want %q", input, got, expected)
	}
}

func TestEscapeXML_UnicodeContent(t *testing.T) {
	// Unicode characters should pass through unchanged
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unicode characters",
			input:    "Hello 世界",
			expected: "Hello 世界",
		},
		{
			name:     "emoji",
			input:    "Hello 👋",
			expected: "Hello 👋",
		},
		{
			name:     "unicode with special chars",
			input:    "Café & Über",
			expected: "Café &amp; Über",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeXML(tt.input)
			if got != tt.expected {
				t.Errorf("escapeXML(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
