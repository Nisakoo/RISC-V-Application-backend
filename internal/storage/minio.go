package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	defaultImageURL = "/defaults/image.svg"
	defaultVideoURL = "/defaults/video.mp4"
)

var ErrInvalidMedia = errors.New("некорректный медиафайл команды RISC-V")

type objectStatClient interface {
	StatObject(ctx context.Context, bucketName string, objectName string, options minio.StatObjectOptions) (minio.ObjectInfo, error)
}

type MinIOStorage struct {
	client  objectStatClient
	writer  *minio.Client
	bucket  string
	baseURL string
}

func NewMinIOStorage(endpoint string, accessKey string, secretKey string, bucket string, useSSL bool, baseURL string) (*MinIOStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("создание клиента MinIO: %w", err)
	}
	return &MinIOStorage{client: client, writer: client, bucket: bucket, baseURL: strings.TrimRight(baseURL, "/")}, nil
}

func (storage *MinIOStorage) Upload(ctx context.Context, kind string, header *multipart.FileHeader) (string, error) {
	if header == nil {
		return "", fmt.Errorf("%w: файл %s не указан", ErrInvalidMedia, kind)
	}
	maximum := int64(5 << 20)
	if kind == "video" {
		maximum = 50 << 20
	}
	if header.Size <= 0 || header.Size > maximum {
		return "", fmt.Errorf("%w: размер файла %s", ErrInvalidMedia, kind)
	}
	file, err := header.Open()
	if err != nil {
		return "", fmt.Errorf("открытие файла %s: %w", kind, err)
	}
	defer file.Close()
	buffer := make([]byte, 512)
	count, err := file.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("чтение файла %s: %w", kind, err)
	}
	contentType := http.DetectContentType(buffer[:count])
	extension := ""
	switch {
	case kind == "images" && contentType == "image/png":
		extension = ".png"
	case kind == "images" && contentType == "image/jpeg":
		extension = ".jpg"
	case kind == "images" && contentType == "image/webp":
		extension = ".webp"
	case kind == "video" && contentType == "video/mp4":
		extension = ".mp4"
	default:
		return "", fmt.Errorf("%w: тип файла %s", ErrInvalidMedia, kind)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return "", fmt.Errorf("повторное чтение файла %s: %w", kind, err)
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("генерация имени файла: %w", err)
	}
	key := kind + "/" + hex.EncodeToString(random) + extension
	if _, err := storage.writer.PutObject(ctx, storage.bucket, key, file, header.Size, minio.PutObjectOptions{ContentType: contentType}); err != nil {
		return "", fmt.Errorf("сохранение файла %s: %w", kind, err)
	}
	return key, nil
}

func (storage *MinIOStorage) Remove(ctx context.Context, key string) error {
	return storage.writer.RemoveObject(ctx, storage.bucket, key, minio.RemoveObjectOptions{})
}

func (storage *MinIOStorage) ImageURL(ctx context.Context, key string) string {
	return storage.availableURL(ctx, key, defaultImageURL)
}

func (storage *MinIOStorage) VideoURL(ctx context.Context, key string) string {
	return storage.availableURL(ctx, key, defaultVideoURL)
}

func (storage *MinIOStorage) availableURL(ctx context.Context, key string, fallbackURL string) string {
	key = strings.TrimLeft(strings.TrimSpace(key), "/")
	if key == "" {
		return fallbackURL
	}
	if _, err := storage.client.StatObject(ctx, storage.bucket, key, minio.StatObjectOptions{}); err != nil {
		return fallbackURL
	}
	return storage.baseURL + "/" + key
}
