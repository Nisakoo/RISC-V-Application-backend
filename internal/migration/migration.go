package migration

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

var versions = []string{
	"001_riscv_commands",
	"002_deleted_draft_constraint",
	"003_methodology_media_and_fields",
	"004_restore_command_length",
	"005_command_fields",
}

//go:embed sql/*.sql
var migrationFiles embed.FS

func Up(ctx context.Context, database *gorm.DB) error {
	return database.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		if err := ensureVersionTable(transaction); err != nil {
			return err
		}
		for _, version := range versions {
			applied, err := isApplied(transaction, version)
			if err != nil {
				return err
			}
			if applied {
				continue
			}
			if err := executeFile(transaction, "sql/"+version+".up.sql"); err != nil {
				return err
			}
			if err := transaction.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version).Error; err != nil {
				return fmt.Errorf("сохранение версии миграции: %w", err)
			}
		}
		return nil
	})
}

func Down(ctx context.Context, database *gorm.DB) error {
	return database.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		if err := ensureVersionTable(transaction); err != nil {
			return err
		}
		for index := len(versions) - 1; index >= 0; index-- {
			version := versions[index]
			applied, err := isApplied(transaction, version)
			if err != nil {
				return err
			}
			if !applied {
				continue
			}
			if err := executeFile(transaction, "sql/"+version+".down.sql"); err != nil {
				return err
			}
			if err := transaction.Exec("DELETE FROM schema_migrations WHERE version = ?", version).Error; err != nil {
				return fmt.Errorf("удаление версии миграции: %w", err)
			}
			return nil
		}
		return nil
	})
}

func ensureVersionTable(database *gorm.DB) error {
	if err := database.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(128) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error; err != nil {
		return fmt.Errorf("создание таблицы миграций: %w", err)
	}
	return nil
}

func isApplied(database *gorm.DB, version string) (bool, error) {
	var count int64
	if err := database.Raw("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count).Error; err != nil {
		return false, fmt.Errorf("проверка версии миграции: %w", err)
	}
	return count > 0, nil
}

func executeFile(database *gorm.DB, name string) error {
	contents, err := migrationFiles.ReadFile(name)
	if err != nil {
		return fmt.Errorf("чтение миграции %s: %w", name, err)
	}
	if strings.TrimSpace(string(contents)) == "" {
		return nil
	}
	if err := database.Exec(string(contents)).Error; err != nil {
		return fmt.Errorf("выполнение миграции %s: %w", name, err)
	}
	return nil
}
