package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Supplier struct {
	ID          primitive.ObjectID `bson:"_id"`
	UserId      int                `bson:"user_id"`
	Active      int8               `bson:"active"`
	Logo        string             `bson:"logo"`
	CompanyName string             `bson:"company_name"`
	// Description          string               `bson:"description"`
	// Images               []string             `bson:"images"`
	// Videos               []string             `bson:"videos"`
	// Address              string               `bson:"address"`
	// Contacts             string               `bson:"contacts"`
	// PersonalTerms        []PersonalTerm       `bson:"personal_terms"`
	// Discounts            Discounts            `bson:"discounts"`
	// Dealers              Dealer               `bson:"dealers"`
	// Moderation           Moderation           `bson:"moderation"`
	// Moderations          []Moderation         `bson:"moderations"`
	// Distributors         []Distributor        `bson:"distributors"`
	// ModeratorLastActions ModeratorLastActions `bson:"moderator_last_actions"`
	// IsSeller             int8                 `bson:"is_seller"`
}

// type PersonalTerm struct {
// 	DealerId            int    `bson:"dealer_id"`
// 	DealerCode          string `bson:"dealer_code"`
// 	Name                string `bson:"name"`
// 	RequestDate         string `bson:"request_date"`
// 	PricesChangeAllowed int8   `bson:"prices_change_allowed"`
// 	PricesChangeDate    string `bson:"prices_change_date"`
// }

// type Discounts struct {
// 	_id                string  `bson:"_id"`
// 	dealer_code        string  `bson:"dealer_code"`
// 	supplier_id        string  `bson:"supplier_id"`
// 	dealer_id          int     `bson:"dealer_id"`
// 	price_type         string  `bson:"price_type"`
// 	category           string  `bson:"category"`
// 	categories         string  `bson:"categories"`
// 	value_percent      float64 `bson:"value_percent"`
// 	brand              string  `bson:"brand,omitempty"`
// 	prices_template    string  `bson:"prices_template"`
// 	updated_at         string  `bson:"updated_at"`
// 	created_at         string  `bson:"created_at"`
// 	category_name_hash string  `bson:"category_name_hash,omitempty"`
// }

// type Dealer struct {
// }

// type Moderation struct {
// }

// type Distributor struct {
// }

// type ModeratorLastActions struct {
// }
