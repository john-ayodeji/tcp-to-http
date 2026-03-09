package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/john-ayodeji/http-from-tcp/internal/headers"
	"github.com/john-ayodeji/http-from-tcp/internal/request"
	"github.com/john-ayodeji/http-from-tcp/internal/response"
	"github.com/john-ayodeji/http-from-tcp/internal/server"
)

const port = 42069

func handler(w *response.Writer, req *request.Request) {
	if strings.HasPrefix(req.RequestLine.RequestTarget, "/httpbin/") {
		proxyHTTPBin(w, req)
		return
	}

	switch req.RequestLine.RequestTarget {
	case "/video":
		body, err := os.ReadFile("assets/vim.mp4")
		if err != nil {
			w.WriteStatusLine(response.StatusInternalServerError)
			h := response.GetDefaultHeaders(0)
			w.WriteHeaders(h)
			w.WriteBody([]byte{})
			return
		}
		w.WriteStatusLine(response.StatusOK)
		h := response.GetDefaultHeaders(len(body))
		h.Set("content-type", "video/mp4")
		w.WriteHeaders(h)
		w.WriteBody(body)
	case "/yourproblem":
		body := []byte(`<html>
  <head>
    <title>400 Bad Request</title>
  </head>
  <body>
    <h1>Bad Request</h1>
    <p>Your request honestly kinda sucked.</p>
  </body>
</html>`)
		w.WriteStatusLine(response.StatusBadRequest)
		h := response.GetDefaultHeaders(len(body))
		h.Set("content-type", "text/html")
		w.WriteHeaders(h)
		w.WriteBody(body)
	case "/myproblem":
		body := []byte(`<html>
  <head>
    <title>500 Internal Server Error</title>
  </head>
  <body>
    <h1>Internal Server Error</h1>
    <p>Okay, you know what? This one is on me.</p>
  </body>
</html>`)
		w.WriteStatusLine(response.StatusInternalServerError)
		h := response.GetDefaultHeaders(len(body))
		h.Set("content-type", "text/html")
		w.WriteHeaders(h)
		w.WriteBody(body)
	default:
		body := []byte(`<html>
  <head>
    <title>200 OK</title>
  </head>
  <body>
    <h1>Success!</h1>
    <p>Your request was an absolute banger.</p>
  </body>
</html>`)
		w.WriteStatusLine(response.StatusOK)
		h := response.GetDefaultHeaders(len(body))
		h.Set("content-type", "text/html")
		w.WriteHeaders(h)
		w.WriteBody(body)
	}
}

func proxyHTTPBin(w *response.Writer, req *request.Request) {
	path := strings.TrimPrefix(req.RequestLine.RequestTarget, "/httpbin")
	url := fmt.Sprintf("https://httpbin.org%s", path)

	resp, err := http.Get(url)
	if err != nil {
		w.WriteStatusLine(response.StatusInternalServerError)
		h := response.GetDefaultHeaders(0)
		w.WriteHeaders(h)
		w.WriteBody([]byte{})
		return
	}
	defer resp.Body.Close()

	w.WriteStatusLine(response.StatusOK)
	h := response.GetDefaultHeaders(0)
	delete(h, "content-length")
	h.Set("transfer-encoding", "chunked")
	h.Set("trailer", "X-Content-SHA256, X-Content-Length")
	h.Set("content-type", "application/json")
	w.WriteHeaders(h)

	var fullBody []byte
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			fmt.Printf("Read %d bytes\n", n)
			fullBody = append(fullBody, buf[:n]...)
			w.WriteChunkedBody(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}
	w.WriteChunkedBodyDone()

	trailers := headers.NewHeaders()
	hash := sha256.Sum256(fullBody)
	trailers.Set("X-Content-SHA256", fmt.Sprintf("%x", hash))
	trailers.Set("X-Content-Length", strconv.Itoa(len(fullBody)))
	w.WriteTrailers(trailers)
}

func main() {
	s, err := server.Serve(port, handler)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer s.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
