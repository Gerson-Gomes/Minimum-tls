package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"minimum-tls/internal/models"
	_ "modernc.org/sqlite"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrEmailExists  = errors.New("email already registered")
	ErrEmptyCart    = errors.New("cart is empty")
)

// Store wraps the SQLite database connection and provides persistence methods.
type Store struct {
	db *sql.DB
}

// Open connects to the SQLite database at path, applies pragmas, and initializes the schema.
func Open(path string) (*Store, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	conn.SetMaxOpenConns(1)

	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA busy_timeout=5000;",
	}
	for _, p := range pragmas {
		if _, err := conn.Exec(p); err != nil {
			conn.Close()
			return nil, fmt.Errorf("apply pragma %q: %w", p, err)
		}
	}

	s := &Store{db: conn}
	if err := s.InitSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return s, nil
}

// Close closes the underlying SQLite connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// InitSchema creates all required tables if they do not already exist.
func (s *Store) InitSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	email TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	avatar TEXT NOT NULL DEFAULT '⚡',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS shoes (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	category TEXT NOT NULL,
	sub_category TEXT NOT NULL DEFAULT '',
	price REAL NOT NULL,
	rating TEXT NOT NULL DEFAULT '★★★★★',
	reviews INTEGER NOT NULL DEFAULT 0,
	badge TEXT NOT NULL DEFAULT '',
	badge_style TEXT NOT NULL DEFAULT '',
	sole_color TEXT NOT NULL DEFAULT '#06b6d4',
	upper_color TEXT NOT NULL DEFAULT '#312e81',
	accent_color TEXT NOT NULL DEFAULT '#ec4899',
	colors_json TEXT NOT NULL DEFAULT '[]',
	trending INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
	promo_code TEXT NOT NULL DEFAULT '',
	discount_rate REAL NOT NULL DEFAULT 0.0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cart_items (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
	shoe_id TEXT NOT NULL,
	name TEXT NOT NULL,
	price REAL NOT NULL,
	size TEXT NOT NULL DEFAULT '9.5',
	qty INTEGER NOT NULL DEFAULT 1,
	UNIQUE(session_id, shoe_id, size)
);

CREATE TABLE IF NOT EXISTS orders (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL,
	user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
	promo_code TEXT NOT NULL DEFAULT '',
	discount_rate REAL NOT NULL DEFAULT 0.0,
	subtotal REAL NOT NULL,
	discount REAL NOT NULL,
	total REAL NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS order_items (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
	shoe_id TEXT NOT NULL,
	name TEXT NOT NULL,
	price REAL NOT NULL,
	size TEXT NOT NULL,
	qty INTEGER NOT NULL
);`
	_, err := s.db.Exec(schema)
	return err
}

// ListShoes returns all shoes, or only trending shoes when trendingOnly is true.
func (s *Store) ListShoes(trendingOnly bool) ([]models.Shoe, error) {
	query := `SELECT id, name, category, sub_category, price, rating, reviews, badge, badge_style, sole_color, upper_color, accent_color, colors_json, trending FROM shoes`
	if trendingOnly {
		query += ` WHERE trending = 1`
	}
	query += ` ORDER BY rowid ASC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shoes []models.Shoe
	for rows.Next() {
		var sh models.Shoe
		var colorsJSON string
		var trendingInt int
		if err := rows.Scan(
			&sh.ID, &sh.Name, &sh.Category, &sh.SubCategory,
			&sh.Price, &sh.Rating, &sh.Reviews, &sh.Badge, &sh.BadgeStyle,
			&sh.SoleColor, &sh.UpperColor, &sh.AccentColor, &colorsJSON, &trendingInt,
		); err != nil {
			return nil, err
		}
		sh.Trending = trendingInt != 0
		if colorsJSON != "" {
			_ = json.Unmarshal([]byte(colorsJSON), &sh.Colors)
		}
		shoes = append(shoes, sh)
	}
	return shoes, rows.Err()
}

// EnsureSession creates a session row if it does not exist.
func (s *Store) EnsureSession(sessionID string) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO sessions (id) VALUES (?)`, sessionID)
	return err
}

// GetSessionCartAndUser loads the cart items, promo state, and associated user for a session.
func (s *Store) GetSessionCartAndUser(sessionID string) (*models.Cart, *models.User, error) {
	if err := s.EnsureSession(sessionID); err != nil {
		return nil, nil, err
	}

	var userID sql.NullString
	cart := &models.Cart{Items: []models.CartItem{}}
	err := s.db.QueryRow(
		`SELECT user_id, promo_code, discount_rate FROM sessions WHERE id = ?`,
		sessionID,
	).Scan(&userID, &cart.PromoCode, &cart.DiscountRate)
	if err != nil {
		return nil, nil, err
	}

	rows, err := s.db.Query(
		`SELECT id, shoe_id, name, price, size, qty FROM cart_items WHERE session_id = ? ORDER BY id ASC`,
		sessionID,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item models.CartItem
		if err := rows.Scan(&item.ID, &item.ShoeID, &item.Name, &item.Price, &item.Size, &item.Qty); err != nil {
			return nil, nil, err
		}
		cart.Items = append(cart.Items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var user *models.User
	if userID.Valid && userID.String != "" {
		u := &models.User{}
		err := s.db.QueryRow(
			`SELECT id, name, email, avatar FROM users WHERE id = ?`,
			userID.String,
		).Scan(&u.ID, &u.Name, &u.Email, &u.Avatar)
		if err == nil {
			user = u
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, nil, err
		}
	}

	return cart, user, nil
}

// AddToCart inserts or increments an item in the session's cart.
func (s *Store) AddToCart(sessionID, shoeID, name string, price float64, size string) error {
	if err := s.EnsureSession(sessionID); err != nil {
		return err
	}
	if size == "" {
		size = "9.5"
	}
	_, err := s.db.Exec(`
		INSERT INTO cart_items (session_id, shoe_id, name, price, size, qty)
		VALUES (?, ?, ?, ?, ?, 1)
		ON CONFLICT(session_id, shoe_id, size) DO UPDATE SET qty = qty + 1
	`, sessionID, shoeID, name, price, size)
	return err
}

// UpdateCartItemQty adjusts the quantity of a cart item and deletes it if qty <= 0.
func (s *Store) UpdateCartItemQty(sessionID string, itemID int64, delta int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`UPDATE cart_items SET qty = qty + ? WHERE id = ? AND session_id = ?`,
		delta, itemID, sessionID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`DELETE FROM cart_items WHERE id = ? AND session_id = ? AND qty <= 0`,
		itemID, sessionID,
	); err != nil {
		return err
	}
	return tx.Commit()
}

// SetCartPromo updates the promo code and discount rate for a session.
func (s *Store) SetCartPromo(sessionID, code string, rate float64) error {
	if err := s.EnsureSession(sessionID); err != nil {
		return err
	}
	_, err := s.db.Exec(
		`UPDATE sessions SET promo_code = ?, discount_rate = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		code, rate, sessionID,
	)
	return err
}

// CreateUser inserts a new user record.
func (s *Store) CreateUser(id, name, email, passwordHash, avatar string) (*models.User, error) {
	_, err := s.db.Exec(
		`INSERT INTO users (id, name, email, password_hash, avatar) VALUES (?, ?, ?, ?, ?)`,
		id, name, email, passwordHash, avatar,
	)
	if err != nil {
		return nil, err
	}
	return &models.User{
		ID:     id,
		Name:   name,
		Email:  email,
		Avatar: avatar,
	}, nil
}

// GetUserByEmail fetches a user and their stored password hash by email.
func (s *Store) GetUserByEmail(email string) (*models.User, string, error) {
	u := &models.User{}
	var hash string
	err := s.db.QueryRow(
		`SELECT id, name, email, password_hash, avatar FROM users WHERE email = ?`,
		email,
	).Scan(&u.ID, &u.Name, &u.Email, &hash, &u.Avatar)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", ErrUserNotFound
	}
	if err != nil {
		return nil, "", err
	}
	return u, hash, nil
}

// AttachUserToSession links a logged-in user to a session.
func (s *Store) AttachUserToSession(sessionID, userID string) error {
	if err := s.EnsureSession(sessionID); err != nil {
		return err
	}
	_, err := s.db.Exec(
		`UPDATE sessions SET user_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		userID, sessionID,
	)
	return err
}

// ClearSessionUser disassociates any user from the session.
func (s *Store) ClearSessionUser(sessionID string) error {
	_, err := s.db.Exec(
		`UPDATE sessions SET user_id = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		sessionID,
	)
	return err
}

// CheckoutSession records an order from the current cart and clears the cart atomically.
func (s *Store) CheckoutSession(orderID, sessionID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var userID sql.NullString
	cart := &models.Cart{}
	if err := tx.QueryRow(
		`SELECT user_id, promo_code, discount_rate FROM sessions WHERE id = ?`,
		sessionID,
	).Scan(&userID, &cart.PromoCode, &cart.DiscountRate); err != nil {
		return err
	}

	rows, err := tx.Query(
		`SELECT shoe_id, name, price, size, qty FROM cart_items WHERE session_id = ?`,
		sessionID,
	)
	if err != nil {
		return err
	}
	for rows.Next() {
		var item models.CartItem
		if err := rows.Scan(&item.ShoeID, &item.Name, &item.Price, &item.Size, &item.Qty); err != nil {
			rows.Close()
			return err
		}
		cart.Items = append(cart.Items, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(cart.Items) == 0 {
		return ErrEmptyCart
	}

	if _, err := tx.Exec(
		`INSERT INTO orders (id, session_id, user_id, promo_code, discount_rate, subtotal, discount, total)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		orderID, sessionID, userID, cart.PromoCode, cart.DiscountRate,
		cart.Subtotal(), cart.Discount(), cart.Total(),
	); err != nil {
		return err
	}

	for _, item := range cart.Items {
		if _, err := tx.Exec(
			`INSERT INTO order_items (order_id, shoe_id, name, price, size, qty) VALUES (?, ?, ?, ?, ?, ?)`,
			orderID, item.ShoeID, item.Name, item.Price, item.Size, item.Qty,
		); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`DELETE FROM cart_items WHERE session_id = ?`, sessionID); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`UPDATE sessions SET promo_code = '', discount_rate = 0.0, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		sessionID,
	); err != nil {
		return err
	}

	return tx.Commit()
}
