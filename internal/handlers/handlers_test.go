package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"minimum-tls/internal/db"
)

func setupTestHandler(t *testing.T) (*Handler, *db.Store) {
	t.Helper()
	store, err := db.Open(filepath.Join(t.TempDir(), "handlers_test.db"))
	if err != nil {
		t.Fatalf("db.Open failed: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	tmpl, err := LoadTemplates(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("LoadTemplates failed: %v", err)
	}
	return New(store, tmpl), store
}

func TestEmptyCatalogAndAuthCartFlow(t *testing.T) {
	h, _ := setupTestHandler(t)

	// 1. Home & Store render empty state cleanly
	for _, path := range []string{"/", "/store"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		if path == "/" {
			h.HomeHandler(rec, req)
		} else {
			h.StoreHandler(rec, req)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s returned %d", path, rec.Code)
		}
	}

	// 2. Signup
	form := url.Values{"name": {"Marcus"}, "email": {"marcus@kinetix.com"}, "password": {"secret123"}}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.SignupAPIHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("signup returned %d: %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected session_id cookie after signup")
	}
	sessCookie := cookies[0]

	// 3. Logout
	reqLogout := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	reqLogout.AddCookie(sessCookie)
	recLogout := httptest.NewRecorder()
	h.LogoutAPIHandler(recLogout, reqLogout)
	if recLogout.Code != http.StatusOK {
		t.Fatalf("logout returned %d", recLogout.Code)
	}

	// 4. Login with wrong password -> 401
	badLogin := url.Values{"email": {"marcus@kinetix.com"}, "password": {"wrong"}}
	reqBad := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(badLogin.Encode()))
	reqBad.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqBad.AddCookie(sessCookie)
	recBad := httptest.NewRecorder()
	h.LoginAPIHandler(recBad, reqBad)
	if recBad.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad password, got %d", recBad.Code)
	}

	// 5. Login with correct password -> 200
	goodLogin := url.Values{"email": {"marcus@kinetix.com"}, "password": {"secret123"}}
	reqGood := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(goodLogin.Encode()))
	reqGood.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqGood.AddCookie(sessCookie)
	recGood := httptest.NewRecorder()
	h.LoginAPIHandler(recGood, reqGood)
	if recGood.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid login, got %d", recGood.Code)
	}

	// 6. Cart add, qty update, promo, checkout
	reqAdd := httptest.NewRequest(http.MethodPost, "/api/cart/add?shoe_id=1&name=Aero&price=189", nil)
	reqAdd.AddCookie(sessCookie)
	recAdd := httptest.NewRecorder()
	h.CartAddHandler(recAdd, reqAdd)
	if recAdd.Code != http.StatusOK || strings.TrimSpace(recAdd.Body.String()) != "1" {
		t.Fatalf("unexpected cart add response: %d %q", recAdd.Code, recAdd.Body.String())
	}

	// Get cart item ID from store
	cart, _, err := h.Store.GetSessionCartAndUser(sessCookie.Value)
	if err != nil || len(cart.Items) != 1 {
		t.Fatalf("expected 1 item in DB cart, got %+v (err=%v)", cart, err)
	}

	reqQty := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/cart/qty?id=%d&delta=1", cart.Items[0].ID), nil)
	reqQty.AddCookie(sessCookie)
	recQty := httptest.NewRecorder()
	h.CartQtyHandler(recQty, reqQty)
	if recQty.Code != http.StatusOK || !strings.Contains(recQty.Body.String(), "Your Shopping Bag (2)") {
		t.Fatalf("unexpected qty response: %d", recQty.Code)
	}

	reqCheckout := httptest.NewRequest(http.MethodPost, "/api/cart/checkout", nil)
	reqCheckout.AddCookie(sessCookie)
	recCheckout := httptest.NewRecorder()
	h.CartCheckoutHandler(recCheckout, reqCheckout)
	if recCheckout.Code != http.StatusOK || !strings.Contains(recCheckout.Body.String(), "Order Confirmed!") {
		t.Fatalf("unexpected checkout response: %d %s", recCheckout.Code, recCheckout.Body.String())
	}
}
