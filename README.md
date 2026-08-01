# ⚡ Minimum-TLS: 14kB 1-TLS-Trip Shoe Store Concept

An ultra-high performance, self-contained e-commerce concept webpage (**KINETIX Footwear**) built to load entirely inside a **single TCP/TLS roundtrip** (< 14,336 bytes Brotli compressed).

![Size Report](https://img.shields.io/badge/Brotli_Size-8.77_kB-10b981?style=for-the-badge&logo=brotli)
![TLS Trip](https://img.shields.io/badge/TLS_Roundtrips-1_Trip-6366f1?style=for-the-badge)
![Dependencies](https://img.shields.io/badge/External_Dependencies-0-06b6d4?style=for-the-badge)

---

## 🎯 The 14kB / 1 TLS Trip Architecture

The initial TCP congestion window (**initcwnd**) of web servers & network routers is standardly **10 MSS** (Maximum Segment Size), which equals **14,336 bytes (14.0 KB)**.

When a webpage's total payload (HTML, inline CSS, vector graphics, icons, and JavaScript) fits strictly under **14,336 bytes Brotli compressed**:
1. The server transmits the **entire application in the very first TCP/TLS packet**.
2. **0ms FOUT / Font Blocking**: Uses system fonts (`system-ui`).
3. **0 External Network Requests**: CSS, JS, and minified vector SVGs are 100% inlined.
4. **Sub-100ms First Contentful Paint (FCP)** worldwide, even on high-latency mobile networks.

---

## ⚡ Size Check Report

```text
=======================================================
 ⚡ 14kB TLS TRIP SIZE CHECK REPORT
=======================================================
 Raw HTML Size:       45,922 bytes (44.85 KB)
 Gzip (Level 9):      10,570 bytes (10.32 KB)
 Brotli (Quality 11): 8,984 bytes (8.77 KB)
 Target Budget Limit: 14,336 bytes (14.00 KB)
-------------------------------------------------------
 ✅ PASS: 8,984 / 14,336 bytes (62.7% used). Budget remaining: 5,352 bytes!
```

---

## 👟 Features & User Flow

- **Hero Banner**: Flagship model showcase with floating specs (*198g Ultra-Light*, *Carbon Plate*, *92% Energy Return*).
- **Real-Time Live Search**: Search by shoe model name or category as you type.
- **Category Filters**: Instant tab filtering (*All*, *Running*, *Basketball*, *Lifestyle*, *Trail*).
- **Dynamic Color Swatches**: Live upper and sole SVG color switching.
- **Product Quick View Modal**: Interactive US size selector (8.0 to 12.0) and specs.
- **Slide-out Shopping Bag Drawer**:
  - Free shipping progress bar ($150 threshold).
  - Quantity adjusters (`+` / `-`).
  - Interactive promo code input (Try entering **`TLS14`** for 10% off!).
  - Dynamic subtotal, tax, discount, and total calculation.
- **Theme Accent Switcher**: Live color accent switching (Indigo, Cyan, Hot Pink, Emerald) via header dot controls.
- **Checkout Flow**: Simulated checkout with toast notification feedback.

---

## 🚀 Quick Start & Testing

### 1. Verify Brotli Compressed Size
```bash
python3 check_size.py
```

### 2. Run Local Brotli Preview Server
```bash
python3 server.py
```
Open `http://localhost:8080` in your browser. Inspect Network DevTools to verify the `Content-Encoding: br` header and 8.98 kB total transfer size.

---

## 📄 License
MIT License. Created as a demonstration of high-efficiency web architecture.
