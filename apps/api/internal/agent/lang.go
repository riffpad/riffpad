package agent

import "unicode"

// DetectLanguage returns "zh" if the prompt contains CJK characters, otherwise "en".
func DetectLanguage(prompt string) string {
	for _, r := range prompt {
		if unicode.Is(unicode.Han, r) {
			return "zh"
		}
	}
	return "en"
}
