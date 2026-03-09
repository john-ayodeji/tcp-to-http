package headers

import (
	"bytes"
	"fmt"
	"strings"
)

type Headers map[string]string

func NewHeaders() Headers {
	return Headers{}
}

func (h Headers) Get(key string) (string, bool) {
	val, ok := h[strings.ToLower(key)]
	return val, ok
}

func (h Headers) Set(key, value string) {
	h[strings.ToLower(key)] = value
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	// Look for a CRLF
	idx := bytes.Index(data, []byte("\r\n"))
	if idx == -1 {
		// Not enough data yet
		return 0, false, nil
	}

	// If the CRLF is at the start, we've reached the end of headers
	if idx == 0 {
		return 2, true, nil
	}

	line := string(data[:idx])
	consumed := idx + 2

	// Split on first colon
	colonIdx := strings.Index(line, ":")
	if colonIdx == -1 {
		return 0, false, fmt.Errorf("invalid header: missing colon in %q", line)
	}

	key := line[:colonIdx]
	value := line[colonIdx+1:]

	// Key must not have trailing spaces (no space between key and colon)
	if len(key) > 0 && key[len(key)-1] == ' ' {
		return 0, false, fmt.Errorf("invalid header: space before colon in %q", line)
	}
	// Key must not have leading spaces
	if len(key) > 0 && key[0] == ' ' {
		return 0, false, fmt.Errorf("invalid header: leading space in key %q", line)
	}
	if len(key) == 0 {
		return 0, false, fmt.Errorf("invalid header: empty key")
	}

	// Validate key contains only valid token characters (RFC 9110 Section 5.1)
	for _, c := range key {
		if !isValidTokenChar(c) {
			return 0, false, fmt.Errorf("invalid header: invalid character %q in key %q", c, key)
		}
	}

	// Trim extra whitespace from value
	value = strings.TrimSpace(value)

	lowerKey := strings.ToLower(key)
	if existing, ok := h[lowerKey]; ok {
		h[lowerKey] = existing + ", " + value
	} else {
		h[lowerKey] = value
	}

	return consumed, false, nil
}

// isValidTokenChar returns true if the rune is a valid HTTP token character
// per RFC 9110 Section 5.6.2 (tchar)
func isValidTokenChar(c rune) bool {
	// token = 1*tchar
	// tchar = "!" / "#" / "$" / "%" / "&" / "'" / "*" / "+" / "-" / "." /
	//         "^" / "_" / "`" / "|" / "~" / DIGIT / ALPHA
	if c >= 'a' && c <= 'z' {
		return true
	}
	if c >= 'A' && c <= 'Z' {
		return true
	}
	if c >= '0' && c <= '9' {
		return true
	}
	switch c {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	}
	return false
}
