package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"sync"

	"minimum-tls/internal/models"
)

type ServerState struct {
	mu       sync.Mutex
	Sessions map[string]*models.Cart
	Users    map[string]*models.User
}

var State = &ServerState{
	Sessions: make(map[string]*models.Cart),
	Users:    make(map[string]*models.User),
}

var templates *template.Template

func InitTemplates() {
	var err error
	templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		panic("Error parsing layout templates: " + err.Error())
	}
	_, err = templates.ParseGlob("templates/partials/*.html")
	if err != nil {
		panic("Error parsing partial templates: " + err.Error())
	}
}

func getOrCreateSession(r *http.Request, w http.ResponseWriter) (*models.Cart, *models.User) {
	State.mu.Lock()
	defer State.mu.Unlock()

	cookie, err := r.Cookie("session_id")
	sessionID := ""
	if err != nil || cookie.Value == "" {
		sessionID = fmt.Sprintf("sess_%d", SystemTimeNano())
		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
		})
	} else {
		sessionID = cookie.Value
	}

	cart, exists := State.Sessions[sessionID]
	if !exists {
		cart = &models.Cart{Items: []models.CartItem{}}
		State.Sessions[sessionID] = cart
	}

	user := State.Users[sessionID]
	return cart, user
}

func RenderPage(w http.ResponseWriter, r *http.Request, activePage string, partialName string) {
	cart, user := getOrCreateSession(r, w)

	// Check if this is an HTMX request requesting only partial main view
	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") == "main-content" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		err := templates.ExecuteTemplate(w, partialName+".html", map[string]interface{}{
			"Cart": cart,
			"User": user,
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
		}
		return
	}

	// Full Page Rendering
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var buf template.HTML
	var b bytesBuffer
	err := templates.ExecuteTemplate(&b, partialName+".html", map[string]interface{}{
		"Cart": cart,
		"User": user,
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	buf = template.HTML(b.String())

	templates.ExecuteTemplate(w, "layout.html", map[string]interface{}{
		"ActivePage": activePage,
		"Content":    buf,
		"Cart":       cart,
		"User":       user,
	})
}

type bytesBuffer struct {
	buf []byte
}

func (b *bytesBuffer) Write(p []byte) (n int, err error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *bytesBuffer) String() string {
	return string(b.buf)
}

func SystemTimeNano() int64 {
	return 1770000000000000000
}

// Page Handlers
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	RenderPage(w, r, "home", "home")
}

func StoreHandler(w http.ResponseWriter, r *http.Request) {
	RenderPage(w, r, "store", "store")
}

func AboutHandler(w http.ResponseWriter, r *http.Request) {
	RenderPage(w, r, "about", "about")
}

func LoginViewHandler(w http.ResponseWriter, r *http.Request) {
	RenderPage(w, r, "login", "login")
}

func SignupViewHandler(w http.ResponseWriter, r *http.Request) {
	RenderPage(w, r, "signup", "signup")
}

// Auth API Handlers
func LoginAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}
	r.ParseForm()
	email := r.FormValue("email")
	name := "Athlete"
	if len(email) > 0 {
		name = email[0:1]
		if idx := len(email); idx > 0 {
			name = email
		}
	}

	State.mu.Lock()
	cookie, _ := r.Cookie("session_id")
	sessID := ""
	if cookie != nil {
		sessID = cookie.Value
	}
	if sessID != "" {
		State.Users[sessID] = &models.User{
			ID:     "usr_1",
			Name:   name,
			Email:  email,
			Avatar: "⚡",
		}
	}
	State.mu.Unlock()

	w.Header().Set("HX-Redirect", "/")
	HomeHandler(w, r)
}

func SignupAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}
	r.ParseForm()
	name := r.FormValue("name")
	email := r.FormValue("email")
	if name == "" {
		name = "Runner"
	}

	State.mu.Lock()
	cookie, _ := r.Cookie("session_id")
	sessID := ""
	if cookie != nil {
		sessID = cookie.Value
	}
	if sessID != "" {
		State.Users[sessID] = &models.User{
			ID:     "usr_new",
			Name:   name,
			Email:  email,
			Avatar: "🚀",
		}
	}
	State.mu.Unlock()

	w.Header().Set("HX-Redirect", "/")
	HomeHandler(w, r)
}

func LogoutAPIHandler(w http.ResponseWriter, r *http.Request) {
	State.mu.Lock()
	cookie, _ := r.Cookie("session_id")
	if cookie != nil {
		delete(State.Users, cookie.Value)
	}
	State.mu.Unlock()

	w.Header().Set("HX-Redirect", "/")
	HomeHandler(w, r)
}

// Cart API Handlers
func CartAddHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}
	cart, _ := getOrCreateSession(r, w)

	shoeID := r.URL.Query().Get("shoe_id")
	name := r.URL.Query().Get("name")
	priceStr := r.URL.Query().Get("price")
	price, _ := strconv.ParseFloat(priceStr, 64)

	State.mu.Lock()
	found := false
	for i, item := range cart.Items {
		if item.ShoeID == shoeID {
			cart.Items[i].Qty++
			found = true
			break
		}
	}
	if !found {
		cart.Items = append(cart.Items, models.CartItem{
			ShoeID: shoeID,
			Name:   name,
			Price:  price,
			Size:   "9.5",
			Qty:    1,
		})
	}
	State.mu.Unlock()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(strconv.Itoa(cart.TotalCount())))
}

func CartDrawerHandler(w http.ResponseWriter, r *http.Request) {
	cart, _ := getOrCreateSession(r, w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.ExecuteTemplate(w, "cart_drawer.html", map[string]interface{}{
		"Cart": cart,
	})
}

func CartQtyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}
	cart, _ := getOrCreateSession(r, w)
	idxStr := r.URL.Query().Get("idx")
	deltaStr := r.URL.Query().Get("delta")
	idx, _ := strconv.Atoi(idxStr)
	delta, _ := strconv.Atoi(deltaStr)

	State.mu.Lock()
	if idx >= 0 && idx < len(cart.Items) {
		cart.Items[idx].Qty += delta
		if cart.Items[idx].Qty <= 0 {
			cart.Items = append(cart.Items[:idx], cart.Items[idx+1:]...)
		}
	}
	State.mu.Unlock()

	CartDrawerHandler(w, r)
}

func CartPromoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}
	r.ParseForm()
	code := r.FormValue("promo")

	cart, _ := getOrCreateSession(r, w)
	State.mu.Lock()
	cart.PromoCode = code
	if code == "TLS14" {
		cart.DiscountRate = 0.10
	} else {
		cart.DiscountRate = 0.0
	}
	State.mu.Unlock()

	CartDrawerHandler(w, r)
}

func CartCheckoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}
	cart, _ := getOrCreateSession(r, w)
	State.mu.Lock()
	cart.Items = []models.CartItem{}
	cart.DiscountRate = 0.0
	cart.PromoCode = ""
	State.mu.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<div style="text-align:center;padding:3rem 1rem">
		<div style="font-size:3rem;margin-bottom:1rem">⚡</div>
		<h2 style="font-size:1.4rem;font-weight:900;margin-bottom:0.5rem">Order Confirmed!</h2>
		<p style="color:var(--muted);font-size:0.9rem">Thank you for testing the KINETIX Go + HTMX platform.</p>
		<button class="btn-primary" style="margin-top:1.5rem" onclick="toggleCart(false)">Done</button>
	</div>`))
}
