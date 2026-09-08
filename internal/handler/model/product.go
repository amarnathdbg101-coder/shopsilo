package model

import (
    "time"
)

type Category struct {
    ID          string     `json:"id"`
    Name        string     `json:"name"`
    Slug        string     `json:"slug"`
    Description string     `json:"description"`
    ImageURL    string     `json:"image_url"`
    ParentID    *string    `json:"parent_id,omitempty"`
    IsActive    bool       `json:"is_active"`
    CreatedAt   time.Time  `json:"created_at"`
    SubCategories []*Category `json:"sub_categories,omitempty"`
}

type Product struct {
    ID              string    `json:"id"`
    ShopID          string    `json:"shop_id"`
    ShopName        string    `json:"shop_name,omitempty"`
    ShopSlug        string    `json:"shop_slug,omitempty"`
    ShopPhone       string    `json:"shop_phone,omitempty"`
    ShopAddress     string    `json:"shop_address,omitempty"`
    ShopCity        string    `json:"shop_city,omitempty"`
    ShopLatitude    *float64  `json:"shop_latitude,omitempty"`
    ShopLongitude   *float64  `json:"shop_longitude,omitempty"`
    CategoryName    string    `json:"category_name,omitempty"`
    Name            string    `json:"name"`
    Slug            string    `json:"slug"`
    Description     string    `json:"description"`
    SKU             string    `json:"sku"`
    Price           float64   `json:"price"`
    CostPrice       float64   `json:"cost_price,omitempty"`
    UnitProfit      float64   `json:"unit_profit,omitempty"`
    ProfitMarginPct float64   `json:"profit_margin_pct,omitempty"`
    ComparePrice    float64   `json:"compare_price,omitempty"`
    CategoryID      string    `json:"category_id"`
    Category        *Category `json:"category,omitempty"`
    Images          []string  `json:"images"`
    Weight          float64   `json:"weight"`
    IsActive        bool      `json:"is_active"`
    IsFeatured      bool      `json:"is_featured"`
    Tags            []string  `json:"tags"`
    Attributes      map[string]interface{} `json:"attributes,omitempty"` // Brand, Model, Size, Color, Gender, Season, etc.
    Inventory       *Inventory `json:"inventory,omitempty"`
    StockQuantity   int        `json:"stock_quantity"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}

type Inventory struct {
    ProductID         string `json:"product_id"`
    Quantity          int    `json:"quantity"`
    ReservedQuantity  int    `json:"reserved_quantity"`
    AvailableQuantity int    `json:"available_quantity"`
    LowStockThreshold int    `json:"low_stock_threshold"`
}