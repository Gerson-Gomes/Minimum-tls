package middleware

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

type bufferedResponseWriter struct {
	http.ResponseWriter
	buf        *bytes.Buffer
	statusCode int
}

func (w *bufferedResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *bufferedResponseWriter) Write(b []byte) (int, error) {
	if w.buf == nil {
		w.buf = &bytes.Buffer{}
	}
	return w.buf.Write(b)
}

func CompressionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptEncoding := r.Header.Get("Accept-Encoding")
		
		bw := &bufferedResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			buf:            &bytes.Buffer{},
		}
		
		next.ServeHTTP(bw, r)

		body := bw.buf.Bytes()
		if len(body) == 0 {
			w.WriteHeader(bw.statusCode)
			return
		}

		// Check if client supports Brotli
		if strings.Contains(acceptEncoding, "br") {
			cmd := exec.Command("brotli", "-q", "11", "-c")
			cmd.Stdin = bytes.NewReader(body)
			var out bytes.Buffer
			cmd.Stdout = &out
			if err := cmd.Run(); err == nil {
				w.Header().Set("Content-Encoding", "br")
				w.Header().Set("Content-Length", strconv.Itoa(out.Len()))
				w.WriteHeader(bw.statusCode)
				w.Write(out.Bytes())
				return
			}
		}

		// Fallback to Gzip if supported
		if strings.Contains(acceptEncoding, "gzip") {
			var out bytes.Buffer
			gzWriter, err := gzip.NewWriterLevel(&out, gzip.BestCompression)
			if err == nil {
				gzWriter.Write(body)
				gzWriter.Close()
				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Set("Content-Length", strconv.Itoa(out.Len()))
				w.WriteHeader(bw.statusCode)
				w.Write(out.Bytes())
				return
			}
		}

		// Raw uncompressed response
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(bw.statusCode)
		w.Write(body)
	})
}
