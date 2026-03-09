package headers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeadersParseSingleValid(t *testing.T) {
	// Test: Valid single header
	headers := NewHeaders()
	data := []byte("Host: localhost:42069\r\n\r\n")
	n, done, err := headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	assert.Equal(t, "localhost:42069", headers["host"])
	assert.Equal(t, 23, n)
	assert.False(t, done)
}

func TestHeadersParseSingleValidExtraWhitespace(t *testing.T) {
	// Test: Valid single header with extra whitespace
	headers := NewHeaders()
	data := []byte("Host:   localhost:42069   \r\n\r\n")
	n, done, err := headers.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, "localhost:42069", headers["host"])
	assert.Equal(t, 28, n)
	assert.False(t, done)
}

func TestHeadersParseTwo(t *testing.T) {
	// Test: Valid 2 headers with existing headers
	headers := NewHeaders()
	headers["accept"] = "*/*"
	data := []byte("Host: localhost:42069\r\nUser-Agent: curl/7.81.0\r\n\r\n")

	// Parse first header
	n, done, err := headers.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, "localhost:42069", headers["host"])
	assert.Equal(t, 23, n)
	assert.False(t, done)

	// Parse second header from remaining data
	n2, done2, err2 := headers.Parse(data[n:])
	require.NoError(t, err2)
	assert.Equal(t, "curl/7.81.0", headers["user-agent"])
	assert.Equal(t, 25, n2)
	assert.False(t, done2)

	// Existing header should still be there
	assert.Equal(t, "*/*", headers["accept"])
}

func TestHeadersParseDone(t *testing.T) {
	// Test: Valid done
	headers := NewHeaders()
	data := []byte("\r\n")
	n, done, err := headers.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.True(t, done)
}

func TestHeadersParseInvalidSpacing(t *testing.T) {
	// Test: Invalid spacing header
	headers := NewHeaders()
	data := []byte("       Host : localhost:42069       \r\n\r\n")
	n, done, err := headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)
}

func TestHeadersParseDuplicateKey(t *testing.T) {
	// Test: Duplicate header key appends value with comma
	headers := NewHeaders()
	headers["set-cookie"] = "cookie1=value1"
	data := []byte("Set-Cookie: cookie2=value2\r\n\r\n")
	n, done, err := headers.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, "cookie1=value1, cookie2=value2", headers["set-cookie"])
	assert.Equal(t, 28, n)
	assert.False(t, done)
}

func TestHeadersParseInvalidCharInKey(t *testing.T) {
	// Test: Invalid character in header key
	headers := NewHeaders()
	data := []byte("H\u00a9st: localhost:42069\r\n\r\n")
	n, done, err := headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)
}
