package redactor

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestRuleFiltering(t *testing.T) {
	config := `
[[rules]]
id = "go-compatible"
description = "Should be kept"
regex = "sk-[a-zA-Z0-9]{32}"

[[rules]]
id = "incompatible-lookaround"
description = "Should be skipped"
regex = "(?<=secret:)[a-z]+"
`
	tmpFile := "test_rules.toml"
	if err := os.WriteFile(tmpFile, []byte(config), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	r, err := New(tmpFile, zerolog.Nop())
	if err != nil {
		t.Fatalf("Failed to create redactor: %v", err)
	}

	if len(r.config.Rules) != 1 {
		t.Errorf("Expected 1 compatible rule, got %d", len(r.config.Rules))
	}
}

func TestRedactRequest(t *testing.T) {
	r := &Redactor{
		config: &Config{
			Rules: []Rule{
				{ID: "test-secret", RawRegex: "SECRET_KEY_[0-9]{5}"},
			},
		},
		logs: zerolog.Nop(),
	}
	if err := r.config.Rules[0].Compile(); err != nil {
		t.Fatalf("Failed to compile rule: %v", err)
	}
	r.detectors = []Detector{NewRegexDetector(r.config.Rules)}

	reqBody := `{"messages": [{"role": "user", "content": "The key is SECRET_KEY_12345"}]}`
	redacted, _ := r.RedactRequest([]byte(reqBody), nil)

	if strings.Contains(string(redacted), "SECRET_KEY_12345") {
		t.Error("Secret not redacted in request")
	}
}

func TestStreamRedactorSlidingWindow(t *testing.T) {
	r := &Redactor{
		config: &Config{
			Rules: []Rule{
				{ID: "split-secret", RawRegex: "DANGER_ZONE"},
			},
		},
		logs: zerolog.Nop(),
	}
	if err := r.config.Rules[0].Compile(); err != nil {
		t.Fatalf("Failed to compile rule: %v", err)
	}
	r.detectors = []Detector{NewRegexDetector(r.config.Rules)}

	// 使用较大窗口以容纳完整占位符
	sr := NewStreamRedactor(r, 30, nil)

	// 模拟敏感词被切分： "DAN" + "GER_ZONE"
	line1 := `data: {"choices":[{"delta":{"content":"DAN"}}]} `
	line2 := `data: {"choices":[{"delta":{"content":"GER_ZONE suffix"}}]} `

	res1 := sr.RedactSSELine([]byte(line1))
	res2 := sr.RedactSSELine([]byte(line2))
	res3 := sr.Flush()

	fullResult := string(res1) + string(res2) + string(res3)

	if strings.Contains(fullResult, "DANGER_ZONE") {
		t.Errorf("Secret leaked: %s", fullResult)
	}
	if !strings.Contains(fullResult, RedactedPlaceholder) {
		t.Errorf("Placeholder missing: %s", fullResult)
	}
}

func TestDetectionLogging(t *testing.T) {
	var buf bytes.Buffer
	r := &Redactor{
		config: &Config{
			Rules: []Rule{
				{ID: "log-rule", Description: "Test Desc", RawRegex: "HIT_ME"},
			},
		},
		logs: zerolog.New(&buf),
	}
	if err := r.config.Rules[0].Compile(); err != nil {
		t.Fatalf("Failed to compile rule: %v", err)
	}
	r.detectors = []Detector{NewRegexDetector(r.config.Rules)}

	r.RedactContent("Text HIT_ME text", map[string]string{"ctx_key": "ctx_val"})

	output := buf.String()
	if !strings.Contains(output, "log-rule") || !strings.Contains(output, "ctx_val") {
		t.Errorf("Audit log incomplete: %s", output)
	}
}

func TestStreamRedactorEdgeCases(t *testing.T) {
	r := &Redactor{
		config: &Config{
			Rules: []Rule{
				{ID: "edge-secret", RawRegex: "MY_PASSWORD"},
			},
		},
		logs: zerolog.Nop(),
	}
	_ = r.config.Rules[0].Compile()
	r.detectors = []Detector{NewRegexDetector(r.config.Rules)}

	sr := NewStreamRedactor(r, 10, nil)

	// Test non-data line
	nonData := []byte("invalid line\n")
	if !bytes.Equal(sr.RedactSSELine(nonData), nonData) {
		t.Error("Should return non-data line untouched")
	}

	// Test DONE line without pending data
	doneLine := []byte("data: [DONE]\n")
	if !bytes.Equal(sr.RedactSSELine(doneLine), doneLine) {
		t.Error("Should return DONE line untouched if no pending")
	}

	// Test DONE line with pending data
	sr.RedactSSELine([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"MY_\"}}]}\n"))
	outDone := sr.RedactSSELine(doneLine)
	if !strings.Contains(string(outDone), "MY_") || !strings.Contains(string(outDone), "[DONE]") {
		t.Error("Should flush pending data before DONE")
	}

	// Test empty content
	sr = NewStreamRedactor(r, 10, nil)
	sr.RedactSSELine([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"MY_\"}}]}\n"))
	emptyContentLine := []byte("data: {\"choices\":[{\"delta\":{}}]}\n")
	outEmpty := sr.RedactSSELine(emptyContentLine)
	if !strings.Contains(string(outEmpty), "MY_") || !strings.Contains(string(outEmpty), "delta\":{}") {
		t.Error("Should flush pending before empty content line")
	}

	// Test invalid JSON
	invalidJSON := []byte("data: {invalid\n")
	if !bytes.Equal(sr.RedactSSELine(invalidJSON), invalidJSON) {
		t.Error("Should return invalid JSON line untouched")
	}

	// Test RedactValue recursively
	val := sr.r.RedactValue([]interface{}{"A_MY_PASSWORD_B", map[string]interface{}{"key": "MY_PASSWORD"}}, nil)
	valJSON, _ := json.Marshal(val)
	if strings.Contains(string(valJSON), "MY_PASSWORD") {
		t.Errorf("RedactValue failed to redact recursively: %s", string(valJSON))
	}

	// Test Config load fallback (JSON instead of TOML)
	jsonConfig := `{"rules": [{"id": "json-rule", "description": "desc", "regex": "JSON_SECRET"}]}`
	tmpFile := "test_rules.json"
	_ = os.WriteFile(tmpFile, []byte(jsonConfig), 0644)
	defer func() { _ = os.Remove(tmpFile) }()
	r2, err := New(tmpFile, zerolog.Nop())
	if err != nil || len(r2.config.Rules) != 1 {
		t.Errorf("Failed to load JSON config: %v", err)
	}
}

func TestRedactorMask(t *testing.T) {
	if mask("1234567") != "****" {
		t.Errorf("Short mask failed")
	}
	if mask("123456789") != "1234...6789" {
		t.Errorf("Long mask failed")
	}
}
