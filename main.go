package main

import (
	"fmt"
	"log"
	"net/http"

	"minimum-tls/internal/handlers"
	"minimum-tls/internal/middleware"
)

func main() {
	handlers.InitTemplates()

	mux := http.NewServeMux()

	// Page routes (supports direct browser GET and HTMX partial GET)
	mux.HandleFunc("GET /", handlers.HomeHandler)
	mux.HandleFunc("GET /store", handlers.StoreHandler)
	mux.HandleFunc("GET /about", handlers.AboutHandler)
	mux.HandleFunc("GET /login", handlers.LoginViewHandler)
	mux.HandleFunc("GET /signup", handlers.SignupViewHandler)

	// Explicit HTMX view partial routes
	mux.HandleFunc("GET /view/home", handlers.HomeHandler)
	mux.HandleFunc("GET /view/store", handlers.StoreHandler)
	mux.HandleFunc("GET /view/about", handlers.AboutHandler)
	mux.HandleFunc("GET /view/login", handlers.LoginViewHandler)
	mux.HandleFunc("GET /view/signup", handlers.SignupViewHandler)

	// Auth API
	mux.HandleFunc("POST /api/auth/login", handlers.LoginAPIHandler)
	mux.HandleFunc("POST /api/auth/signup", handlers.SignupAPIHandler)
	mux.HandleFunc("POST /api/auth/logout", handlers.LogoutAPIHandler)

	// Cart API
	mux.HandleFunc("POST /api/cart/add", handlers.CartAddHandler)
	mux.HandleFunc("GET /api/cart/drawer", handlers.CartDrawerHandler)
	mux.HandleFunc("POST /api/cart/qty", handlers.CartQtyHandler)
	mux.HandleFunc("POST /api/cart/promo", handlers.CartPromoHandler)
	mux.HandleFunc("POST /api/cart/checkout", handlers.CartCheckoutHandler)

	// Wrap in Brotli / Gzip compression middleware
	wrappedHandler := middleware.CompressionMiddleware(mux)

	port := 8080
	fmt.Printf("🚀 KINETIX Go + HTMX Server running on http://localhost:%d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), wrappedHandler))
}
