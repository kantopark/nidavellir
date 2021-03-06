package libs

import "strings"

func IsIn(value string, values []string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

// Lower cases a string and trims its spaces. Used for unique checks
func LowerTrim(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// Upper cases a string and trims its spaces
func UpperTrim(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// Lowers a string, trims its spaces and replaces all spaces in the middle with dashes
// Used to build unique file path
func LowerTrimReplaceSpace(s string) string {
	return strings.Replace(LowerTrim(s), " ", "-", -1)
}

func IsEmptyOrWhitespace(s string) bool {
	return strings.TrimSpace(s) == ""
}
