package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Product — документ коллекции products в MongoDB.
// Это товар поставщика в маркетплейсе.
type Product struct {
	ID          primitive.ObjectID `bson:"_id"`
	SupplierID  primitive.ObjectID `bson:"supplier_id"` // ссылка на поставщика
	ExternalID  string             `bson:"external_id"` // id товара в системе поставщика
	SKU         string             `bson:"sku"`         // артикул
	Name        string             `bson:"name"`
	Description string             `bson:"description,omitempty"`
	Price       float64            `bson:"price"`
	Currency    string             `bson:"currency,omitempty"`
	Quantity    int64              `bson:"quantity"`
	CategoryID  primitive.ObjectID `bson:"category_id,omitempty"`
	Images      []string           `bson:"images,omitempty"`
	IsActive    bool               `bson:"is_active"`

	// Служебные поля аудита
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`

	// Ссылка на импорт, который последним менял товар.
	// Полезно для отладки: «кто записал эту цену?»
	LastImportID string `bson:"last_import_id,omitempty"`
}
