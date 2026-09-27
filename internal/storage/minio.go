package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	defaultImageURL = "/defaults/image.svg"
	defaultVideoURL = "/defaults/video.mp4"
)

type objectStatClient interface {
	StatObject(ctx context.Context, bucketName string, objectName string, options minio.StatObjectOptions) (minio.ObjectInfo, error)
}

type MinIOStorage struct {
	client  objectStatClient
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
