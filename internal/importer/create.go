package importer

import (
	"context"
	"test/internal/models"
)

func (i *Importer) CreateProduct(ctx context.Context, importProduct models.ImportProduct, categories []models.Category) (models.Product, error) {
	product := models.Product{
		Name: "test product ",
	}
	return product, nil
}
