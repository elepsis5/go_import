package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Import — документ коллекции imports: общее состояние импорта поставщика.
// Создаётся PHP-стороной, воркер только двигает статус и счётчики.
type Import struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Type        string             `bson:"type"`
	SupplierID  primitive.ObjectID `bson:"supplier_id"`
	Status      string             `bson:"status"` // StatusQueued / Processing / Completed / Failed
	Total       int64              `bson:"total_count"`
	DoneCount   int64              `bson:"done_count"`
	FailedCount int64              `bson:"failed_count"`
	Error       string             `bson:"error,omitempty"` // последняя ошибка, если была
	CreatedAt   time.Time          `bson:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"`
}
