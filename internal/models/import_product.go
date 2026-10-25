package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportProduct — документ коллекции import_products.
// Создаётся воркером при обработке чанка, дальше PHP двигает статусы валидации.
type ImportProduct struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	ImportID   string             `bson:"import_id"`
	SupplierID string             `bson:"supplier_id"`
	UserID     int32              `bson:"user_id"`
	Index      int32              `bson:"index"`

	// Raw — данные из загрузки поставщика.
	// В Mongo все значения — строки ("430", "210000"), поэтому тип — string.
	Product RawProduct `bson:"product"`

	Article string `bson:"article"`

	// Статус импорта товара: new / update_with_moderation / imported и т. д.
	// Держим как строку — словарь статусов живёт на стороне PHP.
	Status string `bson:"status"`

	// ВАЖНО: инициализировать как []string{}, а не nil — иначе в Mongo
	// запишется null вместо пустого массива, а PHP-код может на этом сломаться.
	Errors   []string `bson:"errors"`
	Warnings []string `bson:"warnings"`

	UpdatedAt                   time.Time  `bson:"updated_at"`
	Validation                  Validation `bson:"validation,omitempty"` // вложенный результат валидации
	ValidationProductsCompleted int32      `bson:"validation_products_completed,omitempty"`
	ImportProductsCompleted     int32      `bson:"import_products_completed,omitempty"`
}

// RawProduct — блок product из документа поставщика.
// Все поля здесь приходят строками из Excel/CSV, поэтому int/float не парсим:
// это зона ответственности валидации (PHP), а не воркера.
type RawProduct struct {
	Name         string   `bson:"name"`
	Description  string   `bson:"description,omitempty"`
	Article      string   `bson:"article"`
	Brand        string   `bson:"brand,omitempty"`
	Category     string   `bson:"category,omitempty"` // "/Кофемашины/" — путь строкой
	EqCategory   string   `bson:"eq_category,omitempty"`
	Country      string   `bson:"country,omitempty"`
	Length       string   `bson:"length,omitempty"`
	Width        string   `bson:"width,omitempty"`
	Height       string   `bson:"height,omitempty"`
	Weight       string   `bson:"weight,omitempty"`
	WeightGross  string   `bson:"weight_gross,omitempty"`
	Power        string   `bson:"power,omitempty"`
	Voltage      string   `bson:"voltage,omitempty"`
	PriceCatalog float64  `bson:"price_catalog,omitempty"`
	PriceDealer  float64  `bson:"price_dealer,omitempty"`
	Currency     string   `bson:"currency,omitempty"`
	Availability string   `bson:"availability,omitempty"`
	DeliveryTime string   `bson:"delivery_time,omitempty"`
	Images       []bson.M `bson:"images,omitempty"` // в примере — одиночная строка с URL
	Files        []bson.M `bson:"files,omitempty"`
	Params       []bson.M `bson:"params,omitempty"` // в примере пустые массивы; если структура известна — замените bson.M на типы
	EqParams     []bson.M `bson:"eq_params,omitempty"`
}

type Validation struct {
	Index    int8     `bson:"index"`
	Article  string   `bson:"article"`
	Product  bson.M   `bson:"product"`
	Warnings []bson.M `bson:"warnings"`
	Status   string   `bson:"status"`
	Errors   []bson.M `bson:"errors"`
	ImportId string   `bson:"import_id"`
}
