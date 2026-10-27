package importer

import (
	"context"
	"test/internal/models"
)

func (i *Importer) UpdateProduct(ctx context.Context, importProduct models.ImportProduct, existingProduct models.Product, categories []models.Category) (models.Product, error) {
	product := models.Product{
		Name: "test product ",
	}
	return product, nil
}
