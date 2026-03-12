package redactor

type RedactionCallback func(match, ruleID, description string) string

type Detector interface {
	Redact(content string, callback RedactionCallback) string
}
