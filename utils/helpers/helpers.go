package helpers

import "strings"

func CheckCommand(message string) (bool, string) {
	// if starts with "!discollama " case insensitive then it's a command
	if strings.HasPrefix(strings.ToLower(message), "!discollama ") {
		return true, strings.TrimSpace(strings.ToLower(strings.TrimPrefix(message, "!discollama ")))
	}
	return false, ""
}
