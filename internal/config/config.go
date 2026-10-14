package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerAddress      string
	DatabaseDSN        string
	MinIOEndpoint      string
	MinIOAccessKey     string
	MinIOSecretKey     string
	MinIOBucket        string
	MinIOUseSSL        bool
	MinIOPublicBaseURL string
	RedisAddress       string
	RedisPassword      string
	JWTSecret          []byte
	JWTTTL             time.Duration
}

// minimumJWTSecretLength задаёт минимальную длину секрета подписи HS256
const minimumJWTSecretLength = 32

func Load() (Config, error) {
	// Локальный файл необязателен: в контейнерах значения передаются окружением.
	_ = godotenv.Load("docker/.env", ".env")

	required := []string{
		"DB_HOST", "DB_PORT", "POSTGRES_DB", "POSTGRES_USER", "POSTGRES_PASSWORD",
		"MINIO_ENDPOINT", "MINIO_ROOT_USER", "MINIO_ROOT_PASSWORD", "MINIO_BUCKET", "MINIO_PUBLIC_BASE_URL",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "JWT_SECRET",
	}
	missing := make([]string, 0)
	for _, name := range required {
		if strings.TrimSpace(os.Getenv(name)) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("не заданы переменные окружения: %s", strings.Join(missing, ", "))
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < minimumJWTSecretLength {
		return Config{}, fmt.Errorf("JWT_SECRET должен содержать не менее %d символов", minimumJWTSecretLength)
	}
	jwtTTL, err := time.ParseDuration(environmentOrDefault("JWT_TTL", "1h"))
	if err != nil || jwtTTL <= 0 {
		return Config{}, fmt.Errorf("JWT_TTL должен быть положительной длительностью, например 1h")
	}

	databaseURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD")),
		Host:   net.JoinHostPort(os.Getenv("DB_HOST"), os.Getenv("DB_PORT")),
		Path:   os.Getenv("POSTGRES_DB"),
	}
	query := databaseURL.Query()
	query.Set("sslmode", environmentOrDefault("DB_SSLMODE", "disable"))
	databaseURL.RawQuery = query.Encode()

	return Config{
		ServerAddress:      environmentOrDefault("SERVER_ADDRESS", ":8080"),
		DatabaseDSN:        databaseURL.String(),
		MinIOEndpoint:      os.Getenv("MINIO_ENDPOINT"),
		MinIOAccessKey:     os.Getenv("MINIO_ROOT_USER"),
		MinIOSecretKey:     os.Getenv("MINIO_ROOT_PASSWORD"),
		MinIOBucket:        os.Getenv("MINIO_BUCKET"),
		MinIOUseSSL:        strings.EqualFold(os.Getenv("MINIO_USE_SSL"), "true"),
		MinIOPublicBaseURL: strings.TrimRight(os.Getenv("MINIO_PUBLIC_BASE_URL"), "/"),
		RedisAddress:       net.JoinHostPort(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
		RedisPassword:      os.Getenv("REDIS_PASSWORD"),
		JWTSecret:          []byte(jwtSecret),
		JWTTTL:             jwtTTL,
	}, nil
}

func environmentOrDefault(name string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
