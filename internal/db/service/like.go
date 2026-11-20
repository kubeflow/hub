package service

import "strings"

// escapeLike escapes the LIKE metacharacters (%, _, \) in s so that it matches literally.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
