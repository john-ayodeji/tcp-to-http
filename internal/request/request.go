package request

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/john-ayodeji/http-from-tcp/internal/headers"
)

type parserState int

const (
	stateInitialized parserState = iota
	stateParsingHeaders
	stateParsingBody
	stateDone
)

type Request struct {
	RequestLine   RequestLine
	Headers       headers.Headers
	Body          []byte
	state         parserState
	contentLength int
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	req := &Request{
		state:   stateInitialized,
		Headers: headers.NewHeaders(),
	}
	buf := make([]byte, 8)
	unparsed := 0 // number of unparsed bytes at the front of buf

	for req.state != stateDone {
		// Grow buffer if it's full
		if unparsed == len(buf) {
			newBuf := make([]byte, len(buf)*2)
			copy(newBuf, buf[:unparsed])
			buf = newBuf
		}

		n, err := reader.Read(buf[unparsed:])
		unparsed += n

		if unparsed > 0 {
			consumed, parseErr := req.parse(buf[:unparsed])
			if parseErr != nil {
				return nil, parseErr
			}
			// Shift unconsumed data to front of buffer
			if consumed > 0 {
				copy(buf, buf[consumed:unparsed])
				unparsed -= consumed
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	if req.state != stateDone {
		return nil, fmt.Errorf("incomplete request")
	}

	return req, nil
}

func (r *Request) parse(data []byte) (int, error) {
	totalBytesParsed := 0
	for r.state != stateDone {
		n, err := r.parseSingle(data[totalBytesParsed:])
		if err != nil {
			return 0, err
		}
		if n == 0 {
			// Need more data
			break
		}
		totalBytesParsed += n
	}
	return totalBytesParsed, nil
}

func (r *Request) parseSingle(data []byte) (int, error) {
	switch r.state {
	case stateInitialized:
		n, err := parseRequestLine(data, &r.RequestLine)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, nil
		}
		r.state = stateParsingHeaders
		return n, nil
	case stateParsingHeaders:
		n, done, err := r.Headers.Parse(data)
		if err != nil {
			return 0, err
		}
		if done {
			r.state = stateParsingBody
			// Parse Content-Length header
			clStr, ok := r.Headers.Get("Content-Length")
			if !ok {
				// No Content-Length, no body to parse
				r.state = stateDone
				return n, nil
			}
			cl, err := strconv.Atoi(clStr)
			if err != nil {
				return 0, fmt.Errorf("invalid Content-Length: %s", clStr)
			}
			r.contentLength = cl
			if cl == 0 {
				r.state = stateDone
			}
		}
		return n, nil
	case stateParsingBody:
		r.Body = append(r.Body, data...)
		if len(r.Body) > r.contentLength {
			return 0, fmt.Errorf("body length exceeds Content-Length: %d > %d", len(r.Body), r.contentLength)
		}
		if len(r.Body) == r.contentLength {
			r.state = stateDone
		}
		return len(data), nil
	case stateDone:
		return 0, fmt.Errorf("error: trying to parse in done state")
	default:
		return 0, fmt.Errorf("unknown parser state")
	}
}

func parseRequestLine(data []byte, rl *RequestLine) (int, error) {
	s := string(data)
	idx := strings.Index(s, "\r\n")
	if idx == -1 {
		// Not enough data yet
		return 0, nil
	}

	line := s[:idx]
	consumed := idx + 2 // include the \r\n

	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid request line: expected 3 parts, got %d", len(parts))
	}

	method := parts[0]
	requestTarget := parts[1]
	versionPart := parts[2]

	// Verify method contains only uppercase alphabetic characters
	if len(method) == 0 {
		return 0, fmt.Errorf("invalid method: empty")
	}
	for _, c := range method {
		if c < 'A' || c > 'Z' {
			return 0, fmt.Errorf("invalid method: %s", method)
		}
	}

	// Verify version is HTTP/1.1
	if !strings.HasPrefix(versionPart, "HTTP/") {
		return 0, fmt.Errorf("invalid HTTP version: %s", versionPart)
	}
	version := strings.TrimPrefix(versionPart, "HTTP/")
	if version != "1.1" {
		return 0, fmt.Errorf("unsupported HTTP version: %s", version)
	}

	rl.Method = method
	rl.RequestTarget = requestTarget
	rl.HttpVersion = version

	return consumed, nil
}
