package models

import "time"

// ==================== МЕДИА ====================

type Media struct {
	ID         int64     `json:"id"`
	EntityType string    `json:"entity_type"`
	EntityID   int64     `json:"entity_id"`
	URL        string    `json:"url"`
	IsMain     bool      `json:"is_main"`
	SortOrder  int       `json:"sort_order"`
	CreatedAt  time.Time `json:"created_at"`
}

// ==================== КАТЕГОРИЯ ====================

type Category struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Slug        string    `json:"slug"`
	Language    string    `json:"language"`
	SortOrder   int       `json:"sort_order"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Images      []Media   `json:"images,omitempty"`
}

type CategoryInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Slug        string `json:"slug"`
	Language    string `json:"language"`
	SortOrder   int    `json:"sort_order"`
	Status      string `json:"status"`
}

// ==================== ПОДКАТЕГОРИЯ ====================

type Subcategory struct {
	ID          int64     `json:"id"`
	CategoryID  int64     `json:"category_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Slug        string    `json:"slug"`
	Language    string    `json:"language"`
	SortOrder   int       `json:"sort_order"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Images      []Media   `json:"images,omitempty"`
}

type SubcategoryInput struct {
	CategoryID  int64  `json:"category_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Slug        string `json:"slug"`
	Language    string `json:"language"`
	SortOrder   int    `json:"sort_order"`
	Status      string `json:"status"`
}

// ==================== ТОВАР ====================

type Product struct {
	ID              int64                    `json:"id"`
	SubcategoryID   int64                    `json:"subcategory_id"`
	Name            string                   `json:"name"`
	Description     string                   `json:"description,omitempty"`
	Content         string                   `json:"content,omitempty"`
	Price           *float64                 `json:"price,omitempty"`
	Slug            string                   `json:"slug"`
	Language        string                   `json:"language"`
	Status          string                   `json:"status"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
	Images          []Media                  `json:"images,omitempty"`
	Characteristics []ProductCharacteristic  `json:"characteristics,omitempty"`
}

type ProductInput struct {
	SubcategoryID int64    `json:"subcategory_id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Content       string   `json:"content"`
	Price         *float64 `json:"price"`
	Slug          string   `json:"slug"`
	Language      string   `json:"language"`
	Status        string   `json:"status"`
}

// ==================== УСЛУГА ====================

type Service struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Content     string    `json:"content,omitempty"`
	Advantages  string    `json:"advantages,omitempty"`
	Slug        string    `json:"slug"`
	Language    string    `json:"language"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Images      []Media   `json:"images,omitempty"`
}

type ServiceInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Advantages  string `json:"advantages"`
	Slug        string `json:"slug"`
	Language    string `json:"language"`
	Status      string `json:"status"`
}

// ==================== ХАРАКТЕРИСТИКИ ====================

type Characteristic struct {
	ID        int64                 `json:"id"`
	Name      string                `json:"name"`
	CreatedAt time.Time             `json:"created_at"`
	Values    []CharacteristicValue `json:"values,omitempty"`
}

type CharacteristicInput struct {
	Name string `json:"name"`
}

type CharacteristicValue struct {
	ID               int64     `json:"id"`
	CharacteristicID int64     `json:"characteristic_id"`
	Value            string    `json:"value"`
	CreatedAt        time.Time `json:"created_at"`
}

type CharacteristicValueInput struct {
	Value string `json:"value"`
}

type ProductCharacteristic struct {
	ID                 int64  `json:"id"`
	ProductID          int64  `json:"product_id"`
	CharacteristicID   int64  `json:"characteristic_id"`
	CharacteristicName string `json:"characteristic_name"`
	ValueID            int64  `json:"value_id"`
	Value              string `json:"value"`
}

type ProductCharacteristicInput struct {
	CharacteristicID int64 `json:"characteristic_id"`
	ValueID          int64 `json:"value_id"`
}

// ==================== АВТОРИЗАЦИЯ ====================

type Admin struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Username  string    `json:"username"`
}

type ChangePasswordInput struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ==================== ОБЩЕЕ ====================

type StatusInput struct {
	Status string `json:"status"`
}
