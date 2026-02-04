package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

type FileService struct {
	client     *minio.Client
	bucketName string
	endpoint   string
}

func NewFileService(client *minio.Client, bucketName, endpoint string) *FileService {
	return &FileService{
		client:     client,
		bucketName: bucketName,
		endpoint:   endpoint,
	}
}

func (s *FileService) UploadFile(
	ctx context.Context,
	file io.Reader,
	fileSize int64,
	originalName string,
	contentType string) (string, error) {

	// 1. Generate unic file_name (UUID + ext)
	ext := filepath.Ext(originalName)
	newFileName := uuid.New().String() + ext

	// 2. Upload in MinIO
	_, err := s.client.PutObject(
		ctx, s.bucketName, newFileName,
		file, fileSize, minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to minio: %w", err)
	}

	// 3. Making url
	// local - http://localhost:9000/images/filename.jpg
	// prod - https://cdn.myapp.com/...
	url := fmt.Sprintf("http://%s/%s/%s", s.endpoint, s.bucketName, newFileName)

	return url, nil
}
