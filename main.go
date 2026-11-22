package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"minimum-tls/internal/db"
	"minimum-tls/internal/handlers"
	"minimum-tls/internal/middleware"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "kinetix.db"
	}

	store, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize SQLite database (%s): %v", dbPath, err)
	}
	defer store.Close()

	tmpl, err := handlers.LoadTemplates(".")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	h := handlers.New(store, tmpl)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", h.HomeHandler)
	mux.HandleFunc("GET /store", h.StoreHandler)
	mux.HandleFunc("GET /about", h.AboutHandler)
	mux.HandleFunc("GET /login", h.LoginViewHandler)
	mux.HandleFunc("GET /signup", h.SignupViewHandler)

	mux.HandleFunc("GET /view/home", h.HomeHandler)
	mux.HandleFunc("GET /view/store", h.StoreHandler)
	mux.HandleFunc("GET /view/about", h.AboutHandler)
	mux.HandleFunc("GET /view/login", h.LoginViewHandler)
	mux.HandleFunc("GET /view/signup", h.SignupViewHandler)

	mux.HandleFunc("POST /api/auth/login", h.LoginAPIHandler)
	mux.HandleFunc("POST /api/auth/signup", h.SignupAPIHandler)
	mux.HandleFunc("POST /api/auth/logout", h.LogoutAPIHandler)

	mux.HandleFunc("POST /api/cart/add", h.CartAddHandler)
	mux.HandleFunc("GET /api/cart/drawer", h.CartDrawerHandler)
	mux.HandleFunc("POST /api/cart/qty", h.CartQtyHandler)
	mux.HandleFunc("POST /api/cart/promo", h.CartPromoHandler)
	mux.HandleFunc("POST /api/cart/checkout", h.CartCheckoutHandler)

	wrappedHandler := middleware.CompressionMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("🚀 KINETIX Go + HTMX Server running on http://localhost:%s (SQLite: %s)\n", port, dbPath)
	log.Fatal(http.ListenAndServe(":"+port, wrappedHandler))
}
