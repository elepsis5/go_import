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
	Product SourceProduct `bson:"product"`

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
type SourceProduct struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Article      string            `json:"article"`
	Brand        string            `json:"brand"`
	Category     string            `json:"category"`
	EqCategory   *string           `json:"eq_category"`
	Country      string            `json:"country"`
	Length       string            `json:"length"`
	Width        string            `json:"width"`
	Height       string            `json:"height"`
	Weight       string            `json:"weight"`
	WeightGross  string            `json:"weight_gross"`
	Power        string            `json:"power"`
	Voltage      string            `json:"voltage"`
	PriceCatalog string            `json:"price_catalog"`
	PriceDealer  string            `json:"price_dealer"`
	Currency     string            `json:"currency"`
	Availability string            `json:"availability"`
	DeliveryTime string            `json:"delivery_time"`
	Images       string            `json:"images"`
	Files        string            `json:"files"`
	Params       map[string]string `json:"params"`
	EqParams     []any             `json:"eq_params"`
}

type Validation struct {
	Index    int8          `bson:"index"`
	Article  string        `bson:"article"`
	Product  ResultProduct `bson:"product"`
	Warnings []bson.M      `bson:"warnings"`
	Status   string        `bson:"status"`
	Errors   []bson.M      `bson:"errors"`
	ImportId string        `bson:"import_id"`
}

type ResultProduct struct {
	PriceCatalog         float64           `json:"price_catalog"`
	PriceDealer          float64           `json:"price_dealer"`
	Images               []string          `json:"images"`
	Files                []string          `json:"files"`
	ImagesAll            []string          `json:"images_all"`
	FilesAll             []string          `json:"files_all"`
	EquipcadID           *int64            `json:"equipcad_id"`
	Equipcad             Equipcad          `json:"equipcad"`
	LastAcceptedDate     *string           `json:"last_accepted_date"`
	Moderation           *Moderation       `json:"moderation"`
	Moderations          []Moderation      `json:"moderations"`
	ModeratorLastActions ModeratorActions  `json:"moderator_last_actions"`
	EqParams             map[string]any    `json:"eq_params"`
	Distributors         []any             `json:"distributors"`
	Offers               []any             `json:"offers"`
	ContentUpdatedAt     *string           `json:"content_updated_at"`
	CommercialUpdatedAt  *string           `json:"commercial_updated_at"`
	Index                int               `json:"index"`
	ImportID             string            `json:"import_id"`
	Active               int               `json:"active"`
	Name                 string            `json:"name"`
	Description          string            `json:"description"`
	Article              string            `json:"article"`
	Brand                string            `json:"brand"`
	CategoryDetected     bool              `json:"category_detected"`
	Category             string            `json:"category"`
	Categories           string            `json:"categories"`
	EqCategory           any               `json:"eq_category"`
	EqCategories         any               `json:"eq_categories"`
	Country              string            `json:"country"`
	Length               int               `json:"length"`
	Width                int               `json:"width"`
	Height               int               `json:"height"`
	Weight               float64           `json:"weight"`
	WeightGross          float64           `json:"weight_gross"`
	Power                float64           `json:"power"`
	Voltage              int               `json:"voltage"`
	Currency             string            `json:"currency"`
	Availability         string            `json:"availability"`
	DeliveryTime         string            `json:"delivery_time"`
	ImagesOrigin         string            `json:"images_origin"`
	FilesOrigin          string            `json:"files_origin"`
	ParamsOrigin         map[string]string `json:"params_origin"`
	Params               map[string]string `json:"params"`
	NeedModeration       int               `json:"need_moderation"`
	OnModeration         OnModeration      `json:"on_moderation"`
}

type Equipcad struct {
	ExportChangesDate *string      `json:"export_changes_date"`
	RecentChanges     *string      `json:"recent_changes"`
	Data              EquipcadData `json:"data"`
	Moderation        *Moderation  `json:"moderation"`
	Moderations       any          `json:"moderations"`
}

type EquipcadData struct {
	Model3D    *string `json:"model_3d"`
	Sketch3D   *string `json:"sketch_3d"`
	Electrical *string `json:"electrical"`
	Water      *string `json:"water"`
	Sewerage   *string `json:"sewerage"`
	Trap       *string `json:"trap"`
	Gas        *string `json:"gas"`
}

type ModeratorActions struct {
	ChangeActivity         *string `json:"change_activity"`
	ChangeFields           *string `json:"change_fields"`
	ChangeModerationStatus *string `json:"change_moderation_status"`
}

type Moderation struct {
	// Поля зависят от фактической структуры moderation.
}

type OnModeration struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Brand       string `json:"brand"`
}
