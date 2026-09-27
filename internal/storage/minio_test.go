package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/minio/minio-go/v7"
)

type fakeObjectStatClient struct {
	existing map[string]bool
}

func (fake fakeObjectStatClient) StatObject(_ context.Context, _ string, objectName string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
	if fake.existing[objectName] {
		return minio.ObjectInfo{Key: objectName}, nil
	}
	return minio.ObjectInfo{}, errors.New("объект отсутствует")
}

func TestMinIOStorageUsesSSRDefaultsForEmptyAndUnavailableMedia(t *testing.T) {
	storage := &MinIOStorage{
		client:  fakeObjectStatClient{existing: map[string]bool{"images/ADD.png": true, "video/ADD.mp4": true}},
		bucket:  "riscv-assets",
		baseURL: "http://localhost:9000/riscv-assets",
	}

	if actual := storage.ImageURL(context.Background(), ""); actual != defaultImageURL {
		t.Fatalf("для пустого изображения получен URL %q", actual)
	}
	if actual := storage.VideoURL(context.Background(), "video/missing.mp4"); actual != defaultVideoURL {
		t.Fatalf("для недоступного видео получен URL %q", actual)
	}
	if actual := storage.ImageURL(context.Background(), "images/ADD.png"); actual != "http://localhost:9000/riscv-assets/images/ADD.png" {
		t.Fatalf("для существующего изображения получен URL %q", actual)
	}
	if actual := storage.VideoURL(context.Background(), "video/ADD.mp4"); actual != "http://localhost:9000/riscv-assets/video/ADD.mp4" {
		t.Fatalf("для существующего видео получен URL %q", actual)
	}
}
