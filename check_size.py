#!/usr/bin/env python3
import os
import sys
import subprocess
import gzip

TARGET_FILE = "index.html"
LIMIT_BYTES = 14336 # 14 KB (14 * 1024)

def check_file():
    if not os.path.exists(TARGET_FILE):
        print(f"Error: {TARGET_FILE} does not exist.")
        sys.exit(1)
        
    with open(TARGET_FILE, "rb") as f:
        content = f.read()
        
    raw_size = len(content)
    gz_content = gzip.compress(content, compresslevel=9)
    gz_size = len(gz_content)
    
    # Use system brotli binary
    res = subprocess.run(["brotli", "-q", "11", "-c", TARGET_FILE], stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    if res.returncode != 0:
        print(f"Error running brotli binary: {res.stderr.decode('utf-8')}")
        sys.exit(1)
        
    br_content = res.stdout
    br_size = len(br_content)
    
    diff = LIMIT_BYTES - br_size
    percent_used = (br_size / LIMIT_BYTES) * 100
    
    print("=" * 55)
    print(" ⚡ 14kB TLS TRIP SIZE CHECK REPORT")
    print("=" * 55)
    print(f" Raw HTML Size:       {raw_size:,} bytes ({raw_size/1024:.2f} KB)")
    print(f" Gzip (Level 9):      {gz_size:,} bytes ({gz_size/1024:.2f} KB)")
    print(f" Brotli (Quality 11): {br_size:,} bytes ({br_size/1024:.2f} KB)")
    print(f" Target Budget Limit: {LIMIT_BYTES:,} bytes ({LIMIT_BYTES/1024:.2f} KB)")
    print("-" * 55)
    
    if br_size <= LIMIT_BYTES:
        print(f" ✅ PASS: {br_size:,} / {LIMIT_BYTES:,} bytes ({percent_used:.1f}% used). Budget remaining: {diff:,} bytes!")
    else:
        print(f" ❌ EXCEEDED BUDGET: Over by {-diff:,} bytes! ({percent_used:.1f}% of limit)")
        sys.exit(1)
        
if __name__ == "__main__":
    check_file()
