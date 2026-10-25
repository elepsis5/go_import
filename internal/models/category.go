package models

type Category struct {
	SupplierId string `bson:"supplier_id"`
	Name       string `bson:"name"`
	NameHash   string `bson:"name_hash"`
	ParentId   string `bson:"parent_id"`
	SyncKey    string `bson:"sync_key"`
	Level      string `bson:"level"`
	Parents    string `bson:"parents"`
}
