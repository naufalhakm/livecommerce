package services

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"time"

	storage_go "github.com/supabase-community/storage-go"
)

type StorageService struct {
	supabaseURL   string
	supabaseKey   string
	bucketName    string
	storageClient *storage_go.Client
}

func NewStorageService() *StorageService {
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_ANON_KEY")
	bucketName := os.Getenv("SUPABASE_STORAGE_BUCKET")
	
	var storageClient *storage_go.Client
	if supabaseURL != "" && supabaseKey != "" {
		storageClient = storage_go.NewClient(supabaseURL+"/storage/v1", supabaseKey, nil)
	}
	
	return &StorageService{
		supabaseURL:   supabaseURL,
		supabaseKey:   supabaseKey,
		bucketName:    bucketName,
		storageClient: storageClient,
	}
}

func (s *StorageService) uploadToSupabase(file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()
	
	// Generate unique filename
	fileName := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
	
	// Upload file using the storage client
	_, err = s.storageClient.UploadFile(s.bucketName, fileName, src)
	if err != nil {
		return "", fmt.Errorf("supabase upload failed: %v", err)
	}
	
	// Get public URL
	publicURL := s.storageClient.GetPublicUrl(s.bucketName, fileName)
	return publicURL.SignedURL, nil
}

func (s *StorageService) UploadFromForm(file *multipart.FileHeader) (string, error) {
	// Use Supabase if configured, otherwise local storage
	if s.storageClient != nil && s.bucketName != "" {
		return s.uploadToSupabase(file)
	}
	return s.uploadLocal(file)
}

func (s *StorageService) uploadLocal(file *multipart.FileHeader) (string, error) {
	// Create uploads directory if it doesn't exist
	uploadsDir := "./uploads"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return "", err
	}
	
	// Generate unique filename
	fileName := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
	filePath := fmt.Sprintf("%s/%s", uploadsDir, fileName)
	
	// Save file
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
	
	// Return public URL
	return fmt.Sprintf("http://100.64.5.96:8080/uploads/%s", fileName), nil
}