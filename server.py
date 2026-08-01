#!/usr/bin/env python3
import http.server
import socketserver
import subprocess
import gzip
import os

PORT = 8080

class BrotliHTTPRequestHandler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/' or self.path == '/index.html':
            if os.path.exists('index.html'):
                with open('index.html', 'rb') as f:
                    content = f.read()
                
                accept_encoding = self.headers.get('Accept-Encoding', '')
                
                if 'br' in accept_encoding:
                    res = subprocess.run(["brotli", "-q", "11", "-c", "index.html"], stdout=subprocess.PIPE)
                    compressed = res.stdout
                    self.send_response(200)
                    self.send_header('Content-Type', 'text/html; charset=utf-8')
                    self.send_header('Content-Encoding', 'br')
                    self.send_header('Content-Length', str(len(compressed)))
                    self.send_header('Cache-Control', 'no-cache')
                    self.end_headers()
                    self.wfile.write(compressed)
                    return
                elif 'gzip' in accept_encoding:
                    compressed = gzip.compress(content, compresslevel=9)
                    self.send_response(200)
                    self.send_header('Content-Type', 'text/html; charset=utf-8')
                    self.send_header('Content-Encoding', 'gzip')
                    self.send_header('Content-Length', str(len(compressed)))
                    self.send_header('Cache-Control', 'no-cache')
                    self.end_headers()
                    self.wfile.write(compressed)
                    return
                else:
                    self.send_response(200)
                    self.send_header('Content-Type', 'text/html; charset=utf-8')
                    self.send_header('Content-Length', str(len(content)))
                    self.send_header('Cache-Control', 'no-cache')
                    self.end_headers()
                    self.wfile.write(content)
                    return

        super().do_GET()

if __name__ == "__main__":
    with socketserver.TCPServer(("", PORT), BrotliHTTPRequestHandler) as httpd:
        print(f"🚀 Server running at http://localhost:{PORT} with Brotli support!")
        httpd.serve_forever()
