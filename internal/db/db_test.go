package db

import (
	"path/filepath"
	"testing"
)

func TestStoreLifecycle(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_empty.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	// 1. Empty catalog on fresh DB
	shoes, err := store.ListShoes(false)
	if err != nil {
		t.Fatalf("ListShoes failed: %v", err)
	}
	if len(shoes) != 0 {
		t.Fatalf("expected 0 shoes in fresh DB, got %d", len(shoes))
	}

	// 2. Create user and look up by email
	u, err := store.CreateUser("usr_1", "Alice", "alice@kinetix.com", "hashedpw", "🚀")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if u.Email != "alice@kinetix.com" {
		t.Fatalf("unexpected email: %s", u.Email)
	}
	gotUser, gotHash, err := store.GetUserByEmail("alice@kinetix.com")
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}
	if gotUser.ID != "usr_1" || gotHash != "hashedpw" {
		t.Fatalf("unexpected user/hash: %+v / %s", gotUser, gotHash)
	}

	// 3. Session, Cart, Promo, and Checkout
	sessID := "sess_test1"
	if err := store.AddToCart(sessID, "shoe_1", "Aero-1", 189.0, "9.5"); err != nil {
		t.Fatalf("AddToCart failed: %v", err)
	}
	if err := store.AddToCart(sessID, "shoe_1", "Aero-1", 189.0, "9.5"); err != nil {
		t.Fatalf("AddToCart increment failed: %v", err)
	}
	if err := store.AttachUserToSession(sessID, "usr_1"); err != nil {
		t.Fatalf("AttachUserToSession failed: %v", err)
	}
	if err := store.SetCartPromo(sessID, "TLS14", 0.10); err != nil {
		t.Fatalf("SetCartPromo failed: %v", err)
	}

	cart, sessUser, err := store.GetSessionCartAndUser(sessID)
	if err != nil {
		t.Fatalf("GetSessionCartAndUser failed: %v", err)
	}
	if sessUser == nil || sessUser.ID != "usr_1" {
		t.Fatalf("expected attached user usr_1, got %+v", sessUser)
	}
	if len(cart.Items) != 1 || cart.Items[0].Qty != 2 {
		t.Fatalf("expected 1 cart item with qty 2, got %+v", cart.Items)
	}
	if cart.DiscountRate != 0.10 || cart.PromoCode != "TLS14" {
		t.Fatalf("unexpected promo state: %+v", cart)
	}

	// Update qty (-1 -> qty 1)
	if err := store.UpdateCartItemQty(sessID, cart.Items[0].ID, -1); err != nil {
		t.Fatalf("UpdateCartItemQty failed: %v", err)
	}
	cart, _, err = store.GetSessionCartAndUser(sessID)
	if err != nil || len(cart.Items) != 1 || cart.Items[0].Qty != 1 {
		t.Fatalf("expected qty 1 after decrement, got %+v (err=%v)", cart, err)
	}

	// Checkout
	if err := store.CheckoutSession("ord_1", sessID); err != nil {
		t.Fatalf("CheckoutSession failed: %v", err)
	}
	cart, _, err = store.GetSessionCartAndUser(sessID)
	if err != nil {
		t.Fatalf("GetSessionCartAndUser after checkout failed: %v", err)
	}
	if len(cart.Items) != 0 || cart.PromoCode != "" || cart.DiscountRate != 0 {
		t.Fatalf("expected empty cart after checkout, got %+v", cart)
	}

	// Verify order persisted
	var orderTotal float64
	var itemCount int
	if err := store.db.QueryRow(`SELECT total FROM orders WHERE id = 'ord_1'`).Scan(&orderTotal); err != nil {
		t.Fatalf("order row missing: %v", err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM order_items WHERE order_id = 'ord_1'`).Scan(&itemCount); err != nil {
		t.Fatalf("order_items query failed: %v", err)
	}
	if orderTotal != 170.10 || itemCount != 1 {
		t.Fatalf("unexpected order total=%v itemCount=%d", orderTotal, itemCount)
	}
}
