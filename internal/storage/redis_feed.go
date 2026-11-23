package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"riscv-lab/internal/domain"
)

// feedPrefix отделяет ключи лент сессий от других ключей Redis
const feedPrefix = "riscv.feed."

// RedisFeedStore хранит текущую порцию ленты каждой сессии
type RedisFeedStore struct {
	client *redis.Client
}

// FeedStore возвращает хранилище лент на том же соединении Redis
func (blacklist *RedisTokenBlacklist) FeedStore() *RedisFeedStore {
	return &RedisFeedStore{client: blacklist.client}
}

// Load читает порцию ленты сессии; found = false, если сессия неизвестна или истекла
func (store *RedisFeedStore) Load(ctx context.Context, sessionID string) (domain.FeedBatch, bool, error) {
	value, err := store.client.Get(ctx, feedPrefix+sessionID).Bytes()
	if errors.Is(err, redis.Nil) {
		return domain.FeedBatch{}, false, nil
	}
	if err != nil {
		return domain.FeedBatch{}, false, fmt.Errorf("чтение ленты из Redis: %w", err)
	}
	var batch domain.FeedBatch
	if err := json.Unmarshal(value, &batch); err != nil {
		return domain.FeedBatch{}, false, fmt.Errorf("разбор ленты из Redis: %w", err)
	}
	return batch, true, nil
}

// Save записывает порцию ленты сессии на ttl
func (store *RedisFeedStore) Save(ctx context.Context, batch domain.FeedBatch, ttl time.Duration) error {
	value, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("запись ленты в Redis: %w", err)
	}
	if err := store.client.Set(ctx, feedPrefix+batch.SessionID, value, ttl).Err(); err != nil {
		return fmt.Errorf("запись ленты в Redis: %w", err)
	}
	return nil
}
