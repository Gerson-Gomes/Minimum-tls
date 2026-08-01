#!/usr/bin/env python3
import os
import sys
import subprocess
import gzip
import urllib.request
import time

LIMIT_BYTES = 14336 # 14 KB (14 * 1024)

def check_go_server():
    print("=" * 60)
    print(" ⚡ PHASE 2: GO SERVER + HTMX 14kB TLS TRIP REPORT")
    print("=" * 60)
    
    proc = subprocess.Popen(["./kinetix-server"])
    time.sleep(1)
    
    try:
        # 1. Full Layout Home Page
        req = urllib.request.Request("http://localhost:8080/", headers={"Accept-Encoding": "br"})
        resp = urllib.request.urlopen(req)
        br_data = resp.read()
        br_size = len(br_data)
        
        req_gz = urllib.request.Request("http://localhost:8080/", headers={"Accept-Encoding": "gzip"})
        resp_gz = urllib.request.urlopen(req_gz)
        gz_data = resp_gz.read()
        gz_size = len(gz_data)
        
        # 2. HTMX Partial View
        req_htmx = urllib.request.Request("http://localhost:8080/view/store", headers={
            "Accept-Encoding": "br",
            "HX-Request": "true",
            "HX-Target": "main-content"
        })
        resp_htmx = urllib.request.urlopen(req_htmx)
        htmx_br_size = len(resp_htmx.read())
        
        diff = LIMIT_BYTES - br_size
        pct = (br_size / LIMIT_BYTES) * 100
        
        print(f" Full Page Brotli (br):  {br_size:,} bytes ({br_size/1024:.2f} KB)")
        print(f" Full Page Gzip (gzip):  {gz_size:,} bytes ({gz_size/1024:.2f} KB)")
        print(f" HTMX Store Partial (br): {htmx_br_size:,} bytes ({htmx_br_size/1024:.2f} KB)")
        print(f" Target 1-TLS Limit:     {LIMIT_BYTES:,} bytes ({LIMIT_BYTES/1024:.2f} KB)")
        print("-" * 60)
        
        if br_size <= LIMIT_BYTES:
            print(f" ✅ PASS: {br_size:,} / {LIMIT_BYTES:,} bytes ({pct:.1f}% used). Budget remaining: {diff:,} bytes!")
        else:
            print(f" ❌ EXCEEDED BUDGET: Over limit by {-diff:,} bytes!")
            sys.exit(1)
            
    finally:
        proc.terminate()

if __name__ == "__main__":
    check_go_server()
