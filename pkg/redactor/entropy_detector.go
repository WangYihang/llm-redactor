package redactor

import (
	"math"
	"regexp"
)

type EntropyDetector struct {
	threshold float64
	minLen    int
	regex     *regexp.Regexp
}

func NewEntropyDetector(threshold float64, minLen int) *EntropyDetector {
	return &EntropyDetector{
		threshold: threshold,
		minLen:    minLen,
		// Token pattern: letters, numbers, and common key symbols (excluding =)
		regex: regexp.MustCompile(`[a-zA-Z0-9\-_+/]{16,}`),
	}
}

var (
	envVarRegex = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
)

func (d *EntropyDetector) Redact(content string, callback RedactionCallback) string {
	return d.regex.ReplaceAllStringFunc(content, func(match string) string {
		if len(match) < d.minLen {
			return match
		}
		// Skip all-uppercase with underscores (likely env vars)
		if envVarRegex.MatchString(match) {
			return match
		}
		if ShannonEntropy(match) > d.threshold {
			return callback(match, "high-entropy", "High entropy token detected")
		}
		return match
	})
}

func ShannonEntropy(data string) float64 {
	if len(data) == 0 {
		return 0
	}
	counts := make(map[rune]int)
	for _, r := range data {
		counts[r]++
	}
	entropy := 0.0
	for _, count := range counts {
		p := float64(count) / float64(len(data))
		entropy -= p * math.Log2(p)
	}
	return entropy
}
