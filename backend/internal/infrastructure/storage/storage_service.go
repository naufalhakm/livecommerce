package storage

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"time"

	"live-shopping-ai/backend/internal/domain/repositories"

	storage_go "github.com/supabase-community/storage-go"
)

type storageService struct {
	supabaseURL   string
	supabaseKey   string
	bucketName    string
	storageClient *storage_go.Client
}

func NewStorageService() repositories.StorageRepository {
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_ANON_KEY")
	bucketName := os.Getenv("SUPABASE_STORAGE_BUCKET")
	
	var storageClient *storage_go.Client
	if supabaseURL != "" && supabaseKey != "" {
		storageClient = storage_go.NewClient(supabaseURL+"/storage/v1", supabaseKey, nil)
	}
	
	return &storageService{
		supabaseURL:   supabaseURL,
		supabaseKey:   supabaseKey,
		bucketName:    bucketName,
		storageClient: storageClient,
	}
}

func (s *storageService) UploadFromForm(file *multipart.FileHeader) (string, error) {
	if s.storageClient != nil && s.bucketName != "" {
		return s.uploadToSupabase(file)
	}
	return s.uploadLocal(file)
}

func (s *storageService) uploadToSupabase(file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()
	
	fileName := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
	
	_, err = s.storageClient.UploadFile(s.bucketName, fileName, src)
	if err != nil {
		return "", fmt.Errorf("supabase upload failed: %v", err)
	}
	
	publicURL := s.storageClient.GetPublicUrl(s.bucketName, fileName)
	return publicURL.SignedURL, nil
}

func (s *storageService) uploadLocal(file *multipart.FileHeader) (string, error) {
	uploadsDir := "./uploads"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return "", err
	}
	
	fileName := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
	filePath := fmt.Sprintf("%s/%s", uploadsDir, fileName)
	
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()
	
	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	
	if _, err = io.Copy(dst, src); err != nil {
		return "", err
	}
	
	return fmt.Sprintf("http://backend:8080/uploads/%s", fileName), nil
}