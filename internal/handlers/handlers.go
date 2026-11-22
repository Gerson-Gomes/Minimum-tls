package handlers

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"minimum-tls/internal/db"
	"minimum-tls/internal/models"
)

// Handler holds dependencies for HTTP and HTMX route handlers.
type Handler struct {
	Store     *db.Store
	Templates *template.Template
}

// New creates a Handler backed by the given SQLite store and parsed templates.
func New(store *db.Store, tmpl *template.Template) *Handler {
	return &Handler{
		Store:     store,
		Templates: tmpl,
	}
}

// LoadTemplates parses layout and partial templates from the given base directory.
func LoadTemplates(baseDir string) (*template.Template, error) {
	tmpl, err := template.ParseGlob(filepath.Join(baseDir, "templates", "*.html"))
	if err != nil {
		return nil, err
	}
	if _, err := tmpl.ParseGlob(filepath.Join(baseDir, "templates", "partials", "*.html")); err != nil {
		return nil, err
	}
	return tmpl, nil
}

func newID(prefix string) string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return prefix + "fallback"
	}
	return prefix + hex.EncodeToString(b[:])
}

func (h *Handler) getOrCreateSessionID(r *http.Request, w http.ResponseWriter) string {
	cookie, err := r.Cookie("session_id")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}
	sessionID := newID("sess_")
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
	})
	return sessionID
}

func (h *Handler) getSessionState(r *http.Request, w http.ResponseWriter) (string, *models.Cart, *models.User, error) {
	sessionID := h.getOrCreateSessionID(r, w)
	cart, user, err := h.Store.GetSessionCartAndUser(sessionID)
	if err != nil {
		return "", nil, nil, err
	}
	return sessionID, cart, user, nil
}

// RenderPage renders either an HTMX partial or the full layout page.
func (h *Handler) RenderPage(w http.ResponseWriter, r *http.Request, activePage, partialName string, shoes []models.Shoe) {
	_, cart, user, err := h.getSessionState(r, w)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Cart":  cart,
		"User":  user,
		"Shoes": shoes,
	}

	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") == "main-content" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := h.Templates.ExecuteTemplate(w, partialName+".html", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	var b bytes.Buffer
	if err := h.Templates.ExecuteTemplate(&b, partialName+".html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.Templates.ExecuteTemplate(w, "layout.html", map[string]interface{}{
		"ActivePage": activePage,
		"Content":    template.HTML(b.String()),
		"Cart":       cart,
		"User":       user,
	})
}

func (h *Handler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	shoes, err := h.Store.ListShoes(true)
	if err != nil {
		http.Error(w, "Failed to load catalog", http.StatusInternalServerError)
		return
	}
	h.RenderPage(w, r, "home", "home", shoes)
}

func (h *Handler) StoreHandler(w http.ResponseWriter, r *http.Request) {
	shoes, err := h.Store.ListShoes(false)
	if err != nil {
		http.Error(w, "Failed to load catalog", http.StatusInternalServerError)
		return
	}
	h.RenderPage(w, r, "store", "store", shoes)
}

func (h *Handler) AboutHandler(w http.ResponseWriter, r *http.Request) {
	h.RenderPage(w, r, "about", "about", nil)
}

func (h *Handler) LoginViewHandler(w http.ResponseWriter, r *http.Request) {
	h.RenderPage(w, r, "login", "login", nil)
}

func (h *Handler) SignupViewHandler(w http.ResponseWriter, r *http.Request) {
	h.RenderPage(w, r, "signup", "signup", nil)
}

func (h *Handler) LoginAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	user, hash, err := h.Store.GetUserByEmail(email)
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	sessID := h.getOrCreateSessionID(r, w)
	if err := h.Store.AttachUserToSession(sessID, user.ID); err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/")
	h.HomeHandler(w, r)
}

func (h *Handler) SignupAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")
	if name == "" {
		name = "Runner"
	}
	if email == "" || password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	user, err := h.Store.CreateUser(newID("usr_"), name, email, string(hash), "🚀")
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, "Email already registered", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	sessID := h.getOrCreateSessionID(r, w)
	if err := h.Store.AttachUserToSession(sessID, user.ID); err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/")
	h.HomeHandler(w, r)
}

func (h *Handler) LogoutAPIHandler(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session_id"); err == nil && cookie.Value != "" {
		_ = h.Store.ClearSessionUser(cookie.Value)
	}
	w.Header().Set("HX-Redirect", "/")
	h.HomeHandler(w, r)
}

func (h *Handler) CartAddHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessID := h.getOrCreateSessionID(r, w)
	shoeID := r.URL.Query().Get("shoe_id")
	name := r.URL.Query().Get("name")
	price, _ := strconv.ParseFloat(r.URL.Query().Get("price"), 64)
	size := r.URL.Query().Get("size")

	if err := h.Store.AddToCart(sessID, shoeID, name, price, size); err != nil {
		http.Error(w, "Failed to add item", http.StatusInternalServerError)
		return
	}

	cart, _, err := h.Store.GetSessionCartAndUser(sessID)
	if err != nil {
		http.Error(w, "Failed to load cart", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(strconv.Itoa(cart.TotalCount())))
}

func (h *Handler) CartDrawerHandler(w http.ResponseWriter, r *http.Request) {
	_, cart, _, err := h.getSessionState(r, w)
	if err != nil {
		http.Error(w, "Failed to load cart", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Templates.ExecuteTemplate(w, "cart_drawer.html", map[string]interface{}{
		"Cart": cart,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) CartQtyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessID := h.getOrCreateSessionID(r, w)
	itemID, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid item id", http.StatusBadRequest)
		return
	}
	delta, _ := strconv.Atoi(r.URL.Query().Get("delta"))

	if err := h.Store.UpdateCartItemQty(sessID, itemID, delta); err != nil {
		http.Error(w, "Failed to update quantity", http.StatusInternalServerError)
		return
	}

	h.CartDrawerHandler(w, r)
}

func (h *Handler) CartPromoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_ = r.ParseForm()
	code := strings.TrimSpace(r.FormValue("promo"))
	rate := 0.0
	if code == "TLS14" {
		rate = 0.10
	}

	sessID := h.getOrCreateSessionID(r, w)
	if err := h.Store.SetCartPromo(sessID, code, rate); err != nil {
		http.Error(w, "Failed to apply promo", http.StatusInternalServerError)
		return
	}

	h.CartDrawerHandler(w, r)
}

func (h *Handler) CartCheckoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessID := h.getOrCreateSessionID(r, w)
	if err := h.Store.CheckoutSession(newID("ord_"), sessID); err != nil {
		if errors.Is(err, db.ErrEmptyCart) {
			http.Error(w, "Your bag is empty", http.StatusBadRequest)
			return
		}
		http.Error(w, "Checkout failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<div style="text-align:center;padding:3rem 1rem">
		<div style="font-size:3rem;margin-bottom:1rem">⚡</div>
		<h2 style="font-size:1.4rem;font-weight:900;margin-bottom:0.5rem">Order Confirmed!</h2>
		<p style="color:var(--muted);font-size:0.9rem">Thank you for testing the KINETIX Go + HTMX platform.</p>
		<button class="btn-primary" style="margin-top:1.5rem" onclick="toggleCart(false)">Done</button>
	</div>`))
}
