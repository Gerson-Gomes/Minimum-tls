# ⚡ Minimum-TLS: Phase 2 Go Server & HTMX Footwear Store

A high-performance e-commerce platform built with a **Golang HTTP server** and **HTMX** for dynamic server-driven UI swapping—engineered to load in a single **1-TLS trip (<14,336 bytes Brotli)**.

![Brotli Size](https://img.shields.io/badge/Brotli_Full_Page-6.39_kB-10b981?style=for-the-badge&logo=brotli)
![HTMX Partial](https://img.shields.io/badge/HTMX_Partial_Payload-1.33_kB-06b6d4?style=for-the-badge)
![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)

---

## 🏗️ Phase 2 Architecture

- **Go HTTP Server (`main.go`, `internal/`)**: Built on Go 1.22+ `net/http` standard routing and session management.
- **Native Brotli Middleware (`internal/middleware/brotli.go`)**: Custom response compression middleware setting `Content-Encoding: br` dynamically for clients supporting Brotli.
- **HTMX Server-Driven Partial Swapping**: Sub-2KB HTML partial responses (`/view/home`, `/view/store`, `/view/about`, `/view/login`, `/view/signup`) updating `<main id="main-content">` with URL history push (`hx-push-url="true"`).
- **Authentication System**: Login & Sign Up screens with cookie session management (`/api/auth/login`, `/api/auth/signup`, `/api/auth/logout`).
- **Server-Side Shopping Cart API**: Session-backed shopping cart (`/api/cart/add`, `/api/cart/drawer`, `/api/cart/qty`, `/api/cart/promo`, `/api/cart/checkout`).

---

## ⚡ Size Audit Report

```text
============================================================
 ⚡ PHASE 2: GO SERVER + HTMX 14kB TLS TRIP REPORT
============================================================
 Full Page Brotli (br):  6,548 bytes (6.39 KB)
 Full Page Gzip (gzip):  7,693 bytes (7.51 KB)
 HTMX Store Partial (br): 1,359 bytes (1.33 KB)
 Target 1-TLS Limit:     14,336 bytes (14.00 KB)
------------------------------------------------------------
 ✅ PASS: 6,548 / 14,336 bytes (45.7% used). Budget remaining: 7,788 bytes!
```

---

## 🚀 Running the Go Server

```bash
# Build the Go server
go build -o kinetix-server .

# Run the server
./kinetix-server
```
Visit `http://localhost:8080` in your browser. Inspect Network DevTools to verify `Content-Encoding: br` and sub-7KB initial transfer.

### Test Size Audit
```bash
python3 check_size.py
```
