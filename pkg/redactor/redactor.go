package redactor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/rs/zerolog"
)

const RedactedPlaceholder = "[REDACTED_SECRET]"

type Redactor struct {
	config *Config
	logs   zerolog.Logger
}

func New(configPath string, logs zerolog.Logger) (*Redactor, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	// Try TOML first (Gitleaks official format)
	if err := toml.Unmarshal(data, &config); err != nil {
		// Fallback to JSON
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config (tried TOML and JSON): %w", err)
		}
	}

	var compatibleRules []Rule
	for _, rule := range config.Rules {
		// Go's regexp engine doesn't support lookaround (?!, ?=, ?<)
		if strings.Contains(rule.RawRegex, "?<") || strings.Contains(rule.RawRegex, "?=") || strings.Contains(rule.RawRegex, "?!") {
			continue
		}
		if err := rule.Compile(); err != nil {
			// Skip invalid/unsupported regex
			continue
		}
		compatibleRules = append(compatibleRules, rule)
	}
	config.Rules = compatibleRules

	return &Redactor{config: &config, logs: logs}, nil
}

func mask(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "..." + s[len(s)-4:]
}

// RedactContent redacts a single string content and logs detections
func (r *Redactor) RedactContent(content string, context map[string]string) string {
	for _, rule := range r.config.Rules {
		// Simple regex replacement
		content = rule.Regex.ReplaceAllStringFunc(content, func(match string) string {
			// Check global allow list
			for _, allow := range r.config.AllowList {
				if match == allow {
					return match
				}
			}

			// LOG DETECTION
			evt := r.logs.Info().
				Str("rule_id", rule.ID).
				Str("description", rule.Description).
				Str("masked_content", mask(match)).
				Int("match_length", len(match))

			for k, v := range context {
				evt.Str(k, v)
			}
			evt.Msg("secret detected")

			return RedactedPlaceholder
		})
	}
	return content
}

// RedactValue recursively traverses a JSON-compatible structure and redacts all string values
func (r *Redactor) RedactValue(v interface{}, context map[string]string) interface{} {
	switch val := v.(type) {
	case string:
		return r.RedactContent(val, context)
	case map[string]interface{}:
		for k, v := range val {
			val[k] = r.RedactValue(v, context)
		}
		return val
	case []interface{}:
		for i, v := range val {
			val[i] = r.RedactValue(v, context)
		}
		return val
	default:
		return v
	}
}

// RedactRequest redacts all string values in a JSON request body
func (r *Redactor) RedactRequest(body []byte, context map[string]string) ([]byte, error) {
	if !json.Valid(body) {
		return body, nil
	}

	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return body, err
	}

	redactedData := r.RedactValue(data, context)
	return json.Marshal(redactedData)
}

// StreamRedactor implements a sliding window redactor for SSE streams
type StreamRedactor struct {
	r       *Redactor
	buffer  []byte
	maxLen  int
	context map[string]string
}

func NewStreamRedactor(r *Redactor, windowSize int, context map[string]string) *StreamRedactor {
	if windowSize <= 0 {
		windowSize = 100
	}
	return &StreamRedactor{
		r:       r,
		maxLen:  windowSize,
		context: context,
	}
}

func (sr *StreamRedactor) extractAndRedact(v interface{}) string {
	switch val := v.(type) {
	case string:
		return sr.r.RedactContent(val, sr.context)
	case map[string]interface{}:
		var fullContent string
		for k, v := range val {
			if k == "content" || k == "text" || k == "thinking" {
				if s, ok := v.(string); ok {
					redacted := sr.r.RedactContent(s, sr.context)
					val[k] = redacted
					fullContent += redacted
				} else {
					fullContent += sr.extractAndRedact(v)
				}
			} else {
				sr.extractAndRedact(v)
			}
		}
		return fullContent
	case []interface{}:
		var fullContent string
		for i, v := range val {
			fullContent += sr.extractAndRedact(v)
			val[i] = sr.redactRecursive(v)
		}
		return fullContent
	}
	return ""
}

func (sr *StreamRedactor) redactRecursive(v interface{}) interface{} {
	return sr.r.RedactValue(v, sr.context)
}

// RedactSSELine processes a single "data: ..." line
func (sr *StreamRedactor) RedactSSELine(line []byte) []byte {
	if !bytes.HasPrefix(line, []byte("data: ")) {
		return line
	}

	rawData := bytes.TrimPrefix(line, []byte("data: "))
	if string(rawData) == "[DONE]" {
		return line
	}

	var data map[string]interface{}
	if err := json.Unmarshal(rawData, &data); err != nil {
		return line
	}

	// Use recursive extraction and redaction
	content := sr.extractAndRedact(data)
	if content == "" {
		return line
	}

	sr.buffer = append(sr.buffer, []byte(content)...)

	if len(sr.buffer) < sr.maxLen {
		sr.r.RedactValue(data, sr.context)
		sr.buffer = []byte(sr.r.RedactContent(string(sr.buffer), sr.context))
	} else {
		toFlush := len(sr.buffer) - sr.maxLen
		sr.buffer = sr.buffer[toFlush:]
		sr.r.RedactValue(data, sr.context)
	}

	newRawData, _ := json.Marshal(data)
	return append([]byte("data: "), newRawData...)
}

func (sr *StreamRedactor) Flush() []byte {
	if len(sr.buffer) == 0 {
		return nil
	}
	redacted := sr.r.RedactContent(string(sr.buffer), sr.context)
	sr.buffer = nil
	return []byte(redacted)
}
