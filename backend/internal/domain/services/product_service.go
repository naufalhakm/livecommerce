package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"live-shopping-ai/backend/internal/domain/entities"
	"live-shopping-ai/backend/internal/domain/repositories"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
)

type ProductService interface {
	GetProducts(ctx context.Context) ([]entities.Product, error)
	GetProduct(ctx context.Context, id int) (*entities.Product, error)
	GetProductsBySellerID(ctx context.Context, sellerID int) ([]entities.Product, error)
	CreateProduct(ctx context.Context, product *entities.Product, images []*multipart.FileHeader) error
	UpdateProduct(ctx context.Context, product *entities.Product) error
	DeleteProduct(ctx context.Context, id int) error
	AddProductImages(ctx context.Context, productID int, images []*multipart.FileHeader) ([]entities.Image, error)
	TrainProductModel(ctx context.Context, productID int) (*entities.TrainingResponse, error)
	PredictProduct(ctx context.Context, productID int, image *multipart.FileHeader) (*entities.PredictionResponse, error)
	PinProduct(ctx context.Context, productID int, sellerID int, similarityScore float64) error
	UnpinProduct(ctx context.Context, productID int, sellerID string) error
	GetPinnedProducts(ctx context.Context, sellerID string) ([]entities.PinnedProduct, error)
	UnpinAllProducts(ctx context.Context, sellerID string) (int64, error)
	TrainAllSellers(ctx context.Context, sellerID string) (*entities.TrainingResponse, error)
	GetTrainingStatus(ctx context.Context, sellerID string) (map[string]interface{}, error)
}

type productService struct {
	productRepo      repositories.ProductRepository
	pinnedRepo       repositories.PinnedProductRepository
	mlRepo           repositories.MLRepository
	storageRepo      repositories.StorageRepository
	mlDatasetBaseDir string
}

func NewProductService(
	productRepo repositories.ProductRepository,
	pinnedRepo repositories.PinnedProductRepository,
	mlRepo repositories.MLRepository,
	storageRepo repositories.StorageRepository,
) ProductService {
	return &productService{
		productRepo:      productRepo,
		pinnedRepo:       pinnedRepo,
		mlRepo:           mlRepo,
		storageRepo:      storageRepo,
		mlDatasetBaseDir: "../ml_service/datasets",
	}
}

func (s *productService) GetProducts(ctx context.Context) ([]entities.Product, error) {
	return s.productRepo.FindAll(ctx)
}

func (s *productService) GetProduct(ctx context.Context, id int) (*entities.Product, error) {
	return s.productRepo.FindByID(ctx, id)
}

func (s *productService) GetProductsBySellerID(ctx context.Context, sellerID int) ([]entities.Product, error) {
	return s.productRepo.FindBySellerID(ctx, sellerID)
}

func (s *productService) CreateProduct(ctx context.Context, product *entities.Product, images []*multipart.FileHeader) error {
	if err := s.productRepo.Create(ctx, product); err != nil {
		return err
	}

	if err := s.createMLDataset(product.ID, product.SellerID, product.Name); err != nil {
		return fmt.Errorf("failed to create ML dataset: %w", err)
	}

	if len(images) > 0 {
		var imageURLs []string
		for _, file := range images {
			imageURL, err := s.storageRepo.UploadFromForm(file)
			if err != nil {
				return fmt.Errorf("failed to upload image: %w", err)
			}
			imageURLs = append(imageURLs, imageURL)
			
			if err := s.saveToMLDataset(product.ID, product.SellerID, file); err != nil {
			}
		}
		
		if err := s.productRepo.AddImages(ctx, product.ID, imageURLs); err != nil {
			return fmt.Errorf("failed to add images: %w", err)
		}
	}

	return nil
}

func (s *productService) UpdateProduct(ctx context.Context, product *entities.Product) error {
	return s.productRepo.Update(ctx, product)
}

func (s *productService) DeleteProduct(ctx context.Context, id int) error {
	return s.productRepo.Delete(ctx, id)
}

func (s *productService) AddProductImages(ctx context.Context, productID int, images []*multipart.FileHeader) ([]entities.Image, error) {
	var imageURLs []string
	var addedImages []entities.Image

	for _, file := range images {
		imageURL, err := s.storageRepo.UploadFromForm(file)
		if err != nil {
			return nil, fmt.Errorf("failed to upload image: %w", err)
		}
		imageURLs = append(imageURLs, imageURL)
	}

	if err := s.productRepo.AddImages(ctx, productID, imageURLs); err != nil {
		return nil, err
	}

	for _, url := range imageURLs {
		addedImages = append(addedImages, entities.Image{
			ProductID: productID,
			ImageURL:  url,
		})
	}

	return addedImages, nil
}

func (s *productService) TrainProductModel(ctx context.Context, productID int) (*entities.TrainingResponse, error) {
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	return s.mlRepo.TrainModel(strconv.Itoa(product.SellerID))
}

func (s *productService) PredictProduct(ctx context.Context, productID int, image *multipart.FileHeader) (*entities.PredictionResponse, error) {
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	src, err := image.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer src.Close()

	imageData := make([]byte, image.Size)
	if _, err := src.Read(imageData); err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	return s.mlRepo.PredictProduct(strconv.Itoa(product.SellerID), imageData)
}

func (s *productService) PinProduct(ctx context.Context, productID int, sellerID int, similarityScore float64) error {
	pinData := &entities.PinnedProduct{
		ProductID:       productID,
		SellerID:        sellerID,
		SimilarityScore: similarityScore,
		IsPinned:        true,
	}
	return s.pinnedRepo.PinProduct(ctx, pinData)
}

func (s *productService) UnpinProduct(ctx context.Context, productID int, sellerID string) error {
	return s.pinnedRepo.UnpinProduct(ctx, productID, sellerID)
}

func (s *productService) GetPinnedProducts(ctx context.Context, sellerID string) ([]entities.PinnedProduct, error) {
	return s.pinnedRepo.FindPinnedBySellerID(ctx, sellerID)
}

func (s *productService) UnpinAllProducts(ctx context.Context, sellerID string) (int64, error) {
	return s.pinnedRepo.UnpinAllProducts(ctx, sellerID)
}

func (s *productService) TrainAllSellers(ctx context.Context, sellerID string) (*entities.TrainingResponse, error) {
	return s.mlRepo.TrainModel(sellerID)
}

func (s *productService) GetTrainingStatus(ctx context.Context, sellerID string) (map[string]interface{}, error) {
	return s.mlRepo.GetTrainingStatus(sellerID)
}

func (s *productService) createMLDataset(productID, sellerID int, productName string) error {
	mlDir := fmt.Sprintf("%s/seller_%d/product_%d", s.mlDatasetBaseDir, sellerID, productID)
	imagesDir := filepath.Join(mlDir, "images")
	
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return err
	}
	
	metadata := map[string]interface{}{
		"product_id":   fmt.Sprintf("product_%d", productID),
		"product_name": productName,
		"seller_id":    fmt.Sprintf("seller_%d", sellerID),
	}
	
	metadataPath := filepath.Join(mlDir, "metadata.json")
	file, err := os.Create(metadataPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	return json.NewEncoder(file).Encode(metadata)
}

func (s *productService) saveToMLDataset(productID, sellerID int, file *multipart.FileHeader) error {
	mlDir := fmt.Sprintf("%s/seller_%d/product_%d/images", s.mlDatasetBaseDir, sellerID, productID)
	filePath := filepath.Join(mlDir, file.Filename)
	
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	
	dst, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer dst.Close()
	
	_, err = io.Copy(dst, src)
	return err
}