package models

// ImportJob — задача из Redis-очереди.
// Именно в таком JSON-виде Laravel кладёт её командой RPUSH go:import:jobs {...}
type ImportJob struct {
	ImportID   string   `json:"import_id"`   // id документа импорта в Mongo (collections.imports)
	SupplierID string   `json:"supplier_id"` // id поставщика, чьи товары импортируем
	ProductIDs []string `json:"product_ids"` // список id товаров в этом чанке
}
