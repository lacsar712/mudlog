package wellbore

import (
	"fmt"
	"strings"
	"unicode"
)

func Validate(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if len(id) > 64 {
		return fmt.Errorf("wellbore id too long")
	}
	for _, r := range id {
		if unicode.IsSpace(r) {
			return fmt.Errorf("wellbore id must not contain spaces")
		}
		ok := unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '/'
		if !ok {
			return fmt.Errorf("wellbore id has illegal character %q", r)
		}
	}
	return nil
}

func Normalize(id string) string {
	return strings.ToUpper(strings.TrimSpace(id))
}
