package redactor

type RegexDetector struct {
	rules []Rule
}

func NewRegexDetector(rules []Rule) *RegexDetector {
	return &RegexDetector{rules: rules}
}

func (d *RegexDetector) Redact(content string, callback RedactionCallback) string {
	for _, rule := range d.rules {
		content = rule.Regex.ReplaceAllStringFunc(content, func(match string) string {
			if len(match) == 0 {
				return match
			}
			return callback(match, rule.ID, rule.Description)
		})
	}
	return content
}
