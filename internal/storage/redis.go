package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// jwtBlacklistPrefix отделяет ключи отозванных JWT от других ключей Redis
const jwtBlacklistPrefix = "riscv.jwt_blacklist."

// RedisTokenBlacklist хранит идентификаторы отозванных JWT до истечения их срока
type RedisTokenBlacklist struct {
	client *redis.Client
}

// NewRedisTokenBlacklist подключается к Redis и проверяет соединение
func NewRedisTokenBlacklist(ctx context.Context, address, password string) (*RedisTokenBlacklist, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         address,
		Password:     password,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("проверка подключения Redis: %w", err)
	}
	return &RedisTokenBlacklist{client: client}, nil
}

// Add помещает идентификатор JWT в blacklist на оставшееся время жизни токена.
// Значением сохраняется логин, чтобы при просмотре Redis было видно владельца токена.
func (blacklist *RedisTokenBlacklist) Add(ctx context.Context, tokenID, login string, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	if err := blacklist.client.Set(ctx, jwtBlacklistPrefix+tokenID, login, ttl).Err(); err != nil {
		return fmt.Errorf("запись JWT в blacklist: %w", err)
	}
	return nil
}

// Contains проверяет, отозван ли JWT
func (blacklist *RedisTokenBlacklist) Contains(ctx context.Context, tokenID string) (bool, error) {
	err := blacklist.client.Get(ctx, jwtBlacklistPrefix+tokenID).Err()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("проверка JWT в blacklist: %w", err)
	}
	return true, nil
}

// Close закрывает соединение с Redis
func (blacklist *RedisTokenBlacklist) Close() error {
	return blacklist.client.Close()
}
