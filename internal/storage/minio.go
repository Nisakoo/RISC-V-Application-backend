package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStorage struct {
	client  *minio.Client
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
	return &MinIOStorage{client: client, bucket: bucket, baseURL: strings.TrimRight(baseURL, "/")}, nil
}

func (storage *MinIOStorage) Put(ctx context.Context, directory string, extension string, contentType string, reader io.Reader, size int64) (string, error) {
	identifier := make([]byte, 16)
	if _, err := rand.Read(identifier); err != nil {
		return "", fmt.Errorf("создание имени медиафайла: %w", err)
	}
	key := strings.Trim(directory, "/") + "/" + hex.EncodeToString(identifier) + extension
	_, err := storage.client.PutObject(ctx, storage.bucket, key, reader, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", fmt.Errorf("загрузка медиафайла %s: %w", key, err)
	}
	return key, nil
}

func (storage *MinIOStorage) Remove(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	if err := storage.client.RemoveObject(ctx, storage.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("удаление медиафайла %s: %w", key, err)
	}
	return nil
}

func (storage *MinIOStorage) URL(key string) string {
	if key == "" {
		return storage.baseURL + "/images/default.svg"
	}
	return storage.baseURL + "/" + strings.TrimLeft(key, "/")
}
