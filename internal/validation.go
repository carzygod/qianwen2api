package internal

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func validateHumanTextField(field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s must be valid UTF-8", field)
	}
	if looksQuestionMarkCorrupted(value) {
		return fmt.Errorf("%s appears to be corrupted into question marks; send JSON as UTF-8 and avoid legacy shell code pages", field)
	}
	return nil
}

func validateVideoRequestText(req VideoGenerationRequest) error {
	if err := validateHumanTextField("prompt", req.Prompt); err != nil {
		return err
	}
	return validateHumanTextField("negative_prompt", req.NegativePrompt)
}

func validateChatRequestText(req ChatRequest) error {
	for idx, message := range req.Messages {
		if err := validateHumanTextField(fmt.Sprintf("messages[%d].content", idx), message.Content); err != nil {
			return err
		}
	}
	return nil
}

func looksQuestionMarkCorrupted(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	total := 0
	questionMarks := 0
	maxRun := 0
	currentRun := 0
	for _, r := range value {
		if unicode.IsSpace(r) {
			continue
		}
		total++
		if r == '?' || r == '？' {
			questionMarks++
			currentRun++
			if currentRun > maxRun {
				maxRun = currentRun
			}
			continue
		}
		currentRun = 0
	}
	return total >= 8 && maxRun >= 4 && questionMarks*3 >= total
}
