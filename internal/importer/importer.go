package importer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"test/internal/models"
	"test/internal/queue"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// Статусы импорта — согласованы со стороной PHP.
// PHP ставит queued перед RPUSH, воркер двигает дальше.
const (
	StatusQueued     = "queued"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

type Importer struct {
	db     *mongo.Database
	logger *zap.Logger
	// dbCategories / eqCategories / supplierID / syncType —
	// те же поля, что обсуждали в execute и createProduct
	// dbCategories     map[string]models.Category
	// eqCategories     map[string]*models.EqCategory
	supplierID       string
	syncType         string
	activeCategories []string
}

func New(db *mongo.Database) (*Importer, error) {
	// logger, err := zap.NewProduction() // или zap.NewDevelopment() для локального дебага
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	return &Importer{db: db, logger: logger}, nil
}

func (i *Importer) Execute(ctx context.Context, job *queue.ImportJob) error {
	i.logger.Info("мы внутри Execute")

	ids := make([]primitive.ObjectID, 0, len(job.ProductIDs))
	for _, idStr := range job.ProductIDs {
		oid, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			i.logger.Error("bad product id", zap.String("id", idStr))
			continue
		}
		ids = append(ids, oid)
	}

	// 1. Забираем записи импорта по _id — чанк уже определён PHP-джобой,
	//    поэтому условной блокировки не нужно (retry учтём отдельно).
	var importProducts []models.ImportProduct
	cur, err := i.db.Collection("import_products").Find(ctx, bson.M{
		"_id":              bson.M{"$in": ids},
		"import_completed": bson.M{"$ne": true},
	})
	if err != nil {
		return err
	}

	err = cur.All(ctx, &importProducts)
	if err != nil {
		return err
	}

	// 2. Загружаем контекст: supplier, категории, eq-категории
	supplier, err := i.loadSupplier(ctx, job.SupplierID)
	if err != nil {
		return err
	}

	importStruct, err := i.loadImport(ctx, job)
	if err != nil {
		return err
	}

	i.supplierID = supplier.ID.Hex()
	i.syncType = importStruct.Type

	categories, err := i.loadCategories(ctx)
	if err != nil {
		return err
	}
	// if err := i.loadEqCategories(ctx, job.ImportID); err != nil {
	// 	return err
	// }

	// 3. Основной цикл — как в PHP: собираем bulk-операции
	var createOps, updateOps []mongo.WriteModel
	completedIDs := make([]primitive.ObjectID, 0, len(importProducts))
	countProducts := 0

	for _, imp := range importProducts {
		switch imp.Validation.Status {
		case "create":
			product, err := i.CreateProduct(ctx, imp, categories)
			if err != nil {
				fields := []zap.Field{
					zap.String("article", imp.Article),
					zap.Error(err),
				}
				i.logger.Error("create failed", fields...)
				continue // не роняем весь чанк из-за одного товара
			}
			createOps = append(createOps,
				mongo.NewInsertOneModel().SetDocument(product))
			completedIDs = append(completedIDs, imp.ID)
			countProducts++

		case "update_with_moderation", "update_without_moderation":
			var existing models.Product
			err := i.db.Collection("products").FindOne(ctx, bson.M{
				"supplier_id": i.supplierID,
				"article":     imp.Article,
			}).Decode(&existing)
			if err == mongo.ErrNoDocuments {
				continue
			} else if err != nil {
				return err
			}
			updated, err := i.UpdateProduct(ctx, imp, existing, categories)
			if err != nil {
				fields := []zap.Field{
					zap.String("article", imp.Article),
					zap.Error(err),
				}
				i.logger.Error("update failed", fields...)
				continue
			}
			updateOps = append(updateOps,
				mongo.NewUpdateOneModel().
					SetFilter(bson.M{"_id": existing.ID}).
					SetUpdate(bson.M{"$set": updated}).
					SetUpsert(false))
			completedIDs = append(completedIDs, imp.ID)
			countProducts++

		case "unchanged":
			// i.handleUnchanged(ctx, imp) // активация категории и т.д.
			completedIDs = append(completedIDs, imp.ID)
			countProducts++
		}
		// failure и status == nil — просто завершаем запись без действий
	}

	// 4. Bulk-запись
	if len(createOps) > 0 {
		res, err := i.db.Collection("products").BulkWrite(ctx, createOps)
		if err != nil {
			return err
		}
		i.logBulkResult("insert", res)
	}
	if len(updateOps) > 0 {
		res, err := i.db.Collection("products").BulkWrite(ctx, updateOps)
		if err != nil {
			return err
		}
		i.logBulkResult("update", res)
	}

	// 5. Помечаем записи обработанными — тоже bulk'ом
	if len(completedIDs) > 0 {
		_, err := i.db.Collection("import_products").BulkWrite(ctx, completedOps(completedIDs))
		if err != nil {
			return err
		}
	}

	// 6. Результат для PHP: пишем в коллекцию импортов
	return i.writeResult(ctx, job, countProducts)
}

func (i *Importer) writeResult(ctx context.Context, job *queue.ImportJob, count int) error {
	_, err := i.db.Collection("imports").UpdateOne(ctx,
		bson.M{"_id": job.ImportID},
		bson.M{"$set": bson.M{
			"sync_status": "done",
			"synced_at":   bson.M{"$currentDate": true},
		}},
		options.Update().SetUpsert(true),
	)
	return err
}

func (i *Importer) loadSupplier(ctx context.Context, id string) (*models.Supplier, error) {
	uid := strings.TrimSpace(id)
	if len(uid) == 0 {
		return nil, errors.New("Suppliers ID must not be empty")
	}
	cur, err := i.db.Collection("suppliers").Find(ctx, bson.M{"_id": uid})
	if err != nil {
		return nil, err
	}

	var supplier models.Supplier
	if err := cur.Decode(&supplier); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("Supplier not found for ID %v: %w", uid, err)
		}
		return nil, err
	}
	return &supplier, nil
}

func (i *Importer) loadImport(ctx context.Context, job *queue.ImportJob) (*models.Import, error) {
	cur, err := i.db.Collection("imports").Find(ctx, bson.M{"_id": job.ImportID})
	if err != nil {
		return nil, err
	}

	// Обязательно закрываем курсор, чтобы не было утечек соединений
	defer cur.Close(ctx)

	var imp models.Import

	if err := cur.Decode(&imp); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("import not found for ID %v: %w", job.ImportID, err)
		}
		return nil, err
	}

	return &imp, nil
}

func (i *Importer) loadCategories(ctx context.Context) ([]models.Category, error) {
	cur, err := i.db.Collection("categories").Find(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var slc []models.Category
	err = cur.All(ctx, &slc)
	if err != nil {
		return nil, err
	}
	return slc, nil
}

func (i *Importer) logBulkResult(op string, res *mongo.BulkWriteResult) {
	// res может быть nil при ошибке до начала bulk-операции
	if res == nil {
		i.logger.Error("bulk operation failed before execution", zap.String("op", op))
		return
	}

	fields := []zap.Field{
		zap.Int64("matched", res.MatchedCount),
		zap.Int64("modified", res.ModifiedCount),
		zap.Int64("upserted", res.UpsertedCount),
		zap.Int64("inserted", res.InsertedCount),
	}

	i.logger.Info("bulk operation completed", fields...)
}

// completedOps формирует массив bulk-операций для пометки записей как обработанных
func completedOps(ids []primitive.ObjectID) []mongo.WriteModel {
	var ops []mongo.WriteModel
	for _, id := range ids {
		// Важно: в MongoDB _id может быть ObjectID или строкой — подстрой тип под свой проект
		filter := bson.M{"_id": id}

		update := bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "status", Value: "processed"},
				{Key: "processed_at", Value: time.Now()},
			}},
		}

		op := mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update)
		ops = append(ops, op)
	}
	return ops
}
