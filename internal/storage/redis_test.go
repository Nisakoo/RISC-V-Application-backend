package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"riscv-lab/internal/config"
)

func TestRedisTokenBlacklistStoresRevokedTokens(t *testing.T) {
	applicationConfig, err := config.Load()
	if err != nil {
		t.Fatalf("загрузка тестовой конфигурации: %v", err)
	}
	ctx := context.Background()
	blacklist, err := NewRedisTokenBlacklist(ctx, applicationConfig.RedisAddress, applicationConfig.RedisPassword)
	if err != nil {
		t.Fatal(err)
	}
	defer blacklist.Close()

	tokenID := "test-" + uuid.NewString()
	defer blacklist.client.Del(ctx, jwtBlacklistPrefix+tokenID)
	if revoked, err := blacklist.Contains(ctx, tokenID); err != nil || revoked {
		t.Fatalf("новый токен уже в blacklist: revoked=%v err=%v", revoked, err)
	}
	if err := blacklist.Add(ctx, tokenID, "rv32_author", time.Minute); err != nil {
		t.Fatal(err)
	}
	if revoked, err := blacklist.Contains(ctx, tokenID); err != nil || !revoked {
		t.Fatalf("отозванный токен не найден: revoked=%v err=%v", revoked, err)
	}
	ttl, err := blacklist.client.TTL(ctx, jwtBlacklistPrefix+tokenID).Result()
	if err != nil || ttl <= 0 || ttl > time.Minute {
		t.Fatalf("время хранения ключа: %v, err=%v", ttl, err)
	}
	if login, err := blacklist.client.Get(ctx, jwtBlacklistPrefix+tokenID).Result(); err != nil || login != "rv32_author" {
		t.Fatalf("значение ключа: %q, err=%v", login, err)
	}
}
