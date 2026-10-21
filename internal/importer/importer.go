package importer

import (
	"context"
	"log"

	"test/internal/models"
	"test/internal/queue"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Importer struct {
	db     *mongo.Database
	logger *log.Logger
	// dbCategories / eqCategories / supplierID / syncType —
	// те же поля, что обсуждали в execute и createProduct
	dbCategories     map[string]models.Category
	eqCategories     map[string]*models.EqCategory
	supplierID       string
	syncType         string
	activeCategories []string
}

func New(db *mongo.Database, logger *log.Logger) *Importer {
	return &Importer{db: db, logger: logger}
}

func (i *Importer) Execute(ctx context.Context, job *queue.ImportJob) error {
	ids := make([]primitive.ObjectID, 0, len(job.ProductIDs))
	for _, idStr := range job.ProductIDs {
		oid, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			i.logger.Printf("bad product id %q: %v", idStr, err)
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

	importStruct, err := i.loadImport(ctx)
	if err != nil {
		return err
	}

	i.supplierID = supplier.ID.Hex()
	i.syncType = importStruct.Type

	if err := i.loadCategories(ctx); err != nil {
		return err
	}
	if err := i.loadEqCategories(ctx, job.ImportID); err != nil {
		return err
	}

	// 3. Основной цикл — как в PHP: собираем bulk-операции
	var createOps, updateOps []mongo.WriteModel
	completedIDs := make([]primitive.ObjectID, 0, len(importProducts))
	countProducts := 0

	for _, imp := range importProducts {
		switch imp.Validation.Status {
		case "create":
			product, err := i.createProduct(ctx, imp)
			if err != nil {
				i.logger.Printf("create failed (article=%s): %v", imp.Article, err)
				continue // не роняем весь чанк из-за одного товара
			}
			createOps = append(createOps,
				mongo.NewInsertOneModel().SetDocument(product.Attrs()))
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
			updated, err := i.updateProduct(ctx, imp, existing)
			if err != nil {
				i.logger.Printf("update failed (article=%s): %v", imp.Article, err)
				continue
			}
			updateOps = append(updateOps,
				mongo.NewUpdateOneModel().
					SetFilter(bson.M{"_id": existing.ID}).
					SetUpdate(bson.M{"$set": updated.Attrs()}).
					SetUpsert(false))
			completedIDs = append(completedIDs, imp.ID)
			countProducts++

		case "unchanged":
			i.handleUnchanged(ctx, imp) // активация категории и т.д.
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
