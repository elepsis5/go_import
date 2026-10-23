package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportJob — задача из Redis-очереди.
// Именно в таком JSON-виде Laravel кладёт её командой RPUSH go:import:jobs {...}
type ImportJob struct {
	ImportID   string   `json:"import_id"`   // id документа импорта в Mongo (collections.imports)
	SupplierID string   `json:"supplier_id"` // id поставщика, чьи товары импортируем
	ProductIDs []string `json:"product_ids"` // список id товаров в этом чанке
}

// Статусы импорта — согласованы со стороной PHP.
// PHP ставит queued перед RPUSH, воркер двигает дальше.
const (
	StatusQueued     = "queued"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

// Product — документ коллекции products в MongoDB.
// Это товар поставщика в маркетплейсе.
type Product struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
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

// Mongo-индексы, которые нужно иметь на коллекции products.
// Не код, а документация для миграций (создаются один раз):
//   {supplier_id: 1, external_id: 1}, unique: true   <- upsert по паре поставщик+внешний id
//   {sku: 1}                                          <- поиск по артикулу
//   {updated_at: -1}                                  <- свежие изменения, отладка

// Import — документ коллекции imports: общее состояние импорта поставщика.
// Создаётся PHP-стороной, воркер только двигает статус и счётчики.
type Import struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	SupplierID  primitive.ObjectID `bson:"supplier_id"`
	Status      string             `bson:"status"` // StatusQueued / Processing / Completed / Failed
	TotalCount  int64              `bson:"total_count"`
	DoneCount   int64              `bson:"done_count"`
	FailedCount int64              `bson:"failed_count"`
	Error       string             `bson:"error,omitempty"` // последняя ошибка, если была
	CreatedAt   time.Time          `bson:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"`
}

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

	UpdatedAt                   time.Time `bson:"updated_at"`
	Validation                  bson.M    `bson:"validation,omitempty"` // вложенный результат валидации
	ValidationProductsCompleted int32     `bson:"validation_products_completed,omitempty"`
	ImportProductsCompleted     int32     `bson:"import_products_completed,omitempty"`
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
	PriceCatalog string   `bson:"price_catalog,omitempty"`
	PriceDealer  string   `bson:"price_dealer,omitempty"`
	Currency     string   `bson:"currency,omitempty"`
	Availability string   `bson:"availability,omitempty"`
	DeliveryTime string   `bson:"delivery_time,omitempty"`
	Images       string   `bson:"images,omitempty"` // в примере — одиночная строка с URL
	Files        string   `bson:"files,omitempty"`
	Params       []bson.M `bson:"params,omitempty"` // в примере пустые массивы; если структура известна — замените bson.M на типы
	EqParams     []bson.M `bson:"eq_params,omitempty"`
}
