package models

type Shoe struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	SubCategory string   `json:"sub_category"`
	Price       float64  `json:"price"`
	Rating      string   `json:"rating"`
	Reviews     int      `json:"reviews"`
	Badge       string   `json:"badge"`
	BadgeStyle  string   `json:"badge_style"`
	SoleColor   string   `json:"sole_color"`
	UpperColor  string   `json:"upper_color"`
	AccentColor string   `json:"accent_color"`
	Colors      []string `json:"colors"`
	Trending    bool     `json:"trending"`
}

type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"-"`
	Avatar   string `json:"avatar"`
}

type CartItem struct {
	ID     int64   `json:"id"`
	ShoeID string  `json:"shoe_id"`
	Name   string  `json:"name"`
	Price  float64 `json:"price"`
	Size   string  `json:"size"`
	Qty    int     `json:"qty"`
}

type Order struct {
	ID           string      `json:"id"`
	SessionID    string      `json:"session_id"`
	UserID       string      `json:"user_id"`
	PromoCode    string      `json:"promo_code"`
	DiscountRate float64     `json:"discount_rate"`
	Subtotal     float64     `json:"subtotal"`
	Discount     float64     `json:"discount"`
	Total        float64     `json:"total"`
	Items        []OrderItem `json:"items"`
}

type OrderItem struct {
	ID      int64   `json:"id"`
	OrderID string  `json:"order_id"`
	ShoeID  string  `json:"shoe_id"`
	Name    string  `json:"name"`
	Price   float64 `json:"price"`
	Size    string  `json:"size"`
	Qty     int     `json:"qty"`
}

type Cart struct {
	Items        []CartItem `json:"items"`
	DiscountRate float64    `json:"discount_rate"`
	PromoCode    string     `json:"promo_code"`
}

func (c *Cart) Subtotal() float64 {
	var total float64
	for _, item := range c.Items {
		total += item.Price * float64(item.Qty)
	}
	return total
}

func (c *Cart) Discount() float64 {
	return c.Subtotal() * c.DiscountRate
}

func (c *Cart) Total() float64 {
	sub := c.Subtotal() - c.Discount()
	if sub < 0 {
		return 0
	}
	return sub
}

func (c *Cart) TotalCount() int {
	var count int
	for _, item := range c.Items {
		count += item.Qty
	}
	return count
}
