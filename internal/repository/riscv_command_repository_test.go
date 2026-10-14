package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"riscv-lab/internal/config"
	"riscv-lab/internal/domain"
)

func TestRiscVCommandRepositoryQueriesPostgreSQL(t *testing.T) {
	repository, rollback := testRepository(t)
	defer rollback()
	ctx := context.Background()
	var creatorID int64
	login := fmt.Sprintf("lab2_query_%d", time.Now().UnixNano())
	if err := repository.db.WithContext(ctx).Raw(
		"INSERT INTO users (login, password_hash, role) VALUES (?, ?, ?) RETURNING id",
		login, "not-used-in-lab-2", "moderator",
	).Scan(&creatorID).Error; err != nil {
		t.Fatal(err)
	}
	if err := repository.db.WithContext(ctx).Exec(`
		INSERT INTO riscv_commands (
			id, mnemonic, description, operand_count, length_bits,
			image_key, video_key, status, creator_id, published_at
		) VALUES
			(-2002, 'TESTU1', 'Тестовая команда для проверки чтения из PostgreSQL.', 2, 32, 'images/default.svg', 'video/ADD.mp4', 'published', ?, CURRENT_TIMESTAMP),
			(-2001, 'TESTU2', 'Тестовая команда для проверки порядка ленты PostgreSQL.', 2, 32, 'images/default.svg', 'video/ADD.mp4', 'published', ?, CURRENT_TIMESTAMP)
	`, creatorID, creatorID).Error; err != nil {
		t.Fatal(err)
	}
	if err := repository.db.WithContext(ctx).Exec(
		"INSERT INTO riscv_command_likes (user_id, riscv_command_id) VALUES (?, ?)", creatorID, -2002,
	).Error; err != nil {
		t.Fatal(err)
	}

	command, err := repository.GetByID(ctx, -2002)
	if err != nil {
		t.Fatal(err)
	}
	if command.Mnemonic != "TESTU1" || command.LikeCount != 1 {
		t.Fatalf("получена команда %+v, ожидалась TESTU1 с одним лайком", command)
	}

	commands, err := repository.GetVisible(ctx, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !hasCommandID(commands, -2002) || !hasCommandID(commands, -2001) {
		t.Fatalf("в выборке отсутствуют тестовые команды: %+v", commands)
	}

	next, err := repository.GetNextVisible(ctx, -2002)
	if err != nil {
		t.Fatal(err)
	}
	if next.ID != -2001 {
		t.Fatalf("после TESTU1 получен ID %d, ожидался -2001", next.ID)
	}
}

func hasCommandID(commands []domain.RiscVCommand, expectedID int64) bool {
	for _, command := range commands {
		if command.ID == expectedID {
			return true
		}
	}
	return false
}

func TestRiscVCommandRepositoryDraftLifecycle(t *testing.T) {
	repository, rollback := testRepository(t)
	defer rollback()
	ctx := context.Background()

	var creatorID int64
	login := fmt.Sprintf("lab2_test_%d", time.Now().UnixNano())
	if err := repository.db.WithContext(ctx).Raw(
		"INSERT INTO users (login, password_hash, role) VALUES (?, ?, ?) RETURNING id",
		login, "not-used-in-lab-2", "moderator",
	).Scan(&creatorID).Error; err != nil {
		t.Fatal(err)
	}

	draft, err := repository.CreateDraft(ctx, creatorID, domain.CreateRiscVCommandDraft{
		Mnemonic: "XOR", ImageKey: "images/missing/xor.png", VideoKey: "video/missing/xor.mp4",
	})
	if err != nil {
		t.Fatal(err)
	}
	if draft.Status != domain.RiscVCommandDraft || draft.CreatorID != creatorID {
		t.Fatalf("создан некорректный черновик: %+v", draft)
	}
	if draft.ImageKey != "images/missing/xor.png" || draft.VideoKey != "video/missing/xor.mp4" {
		t.Fatalf("ключи медиа черновика сохранены неверно: %+v", draft)
	}
	var missingFields int64
	if err := repository.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM riscv_commands
		WHERE id = ? AND description IS NULL AND operand_count IS NULL
		AND length_bits IS NULL AND published_at IS NULL
	`, draft.ID).Scan(&missingFields).Error; err != nil {
		t.Fatal(err)
	}
	if missingFields != 1 {
		t.Fatal("поля нового черновика должны быть NULL до публикации")
	}
	// Ошибка вставки прерывает транзакцию теста, поэтому проверка выполняется внутри точки сохранения
	repository.db.SavePoint("media_keys")
	var postgresError *pgconn.PgError
	if _, err := repository.CreateDraft(ctx, creatorID, domain.CreateRiscVCommandDraft{Mnemonic: "AND"}); !errors.As(err, &postgresError) || postgresError.ConstraintName != "riscv_commands_media_keys_not_empty" {
		t.Fatalf("черновик без url медиа должен нарушать ограничение схемы: %v", err)
	}
	repository.db.RollbackTo("media_keys")
	published, err := repository.PublishDraft(ctx, draft.ID, creatorID, domain.PublishRiscVCommandDraft{
		Description:  "Выполняет побитовое исключающее ИЛИ для двух исходных регистров.",
		OperandCount: 3,
		LengthBits:   32,
	})
	if err != nil {
		t.Fatal(err)
	}
	if published.Status != domain.RiscVCommandPublished || published.PublishedAt == nil {
		t.Fatalf("команда не опубликована: %+v", published)
	}

	if err := repository.SoftDelete(ctx, published.ID); err != nil {
		t.Fatal(err)
	}
	deleted, err := repository.GetByID(ctx, published.ID)
	if err != nil {
		t.Fatal(err)
	}
	if deleted.Status != domain.RiscVCommandDeleted {
		t.Fatalf("получен статус %q, ожидался deleted", deleted.Status)
	}

	if _, err := repository.CreateDraft(ctx, creatorID, domain.CreateRiscVCommandDraft{Mnemonic: "SLT", ImageKey: "images/missing/slt.png", VideoKey: "video/missing/slt.mp4"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CreateDraft(ctx, creatorID, domain.CreateRiscVCommandDraft{Mnemonic: "SLL", ImageKey: "images/missing/sll.png", VideoKey: "video/missing/sll.mp4"}); err != ErrDraftAlreadyExists {
		t.Fatalf("получена ошибка %v, ожидалась %v", err, ErrDraftAlreadyExists)
	}
}

func testRepository(t *testing.T) (*RiscVCommandRepository, func()) {
	t.Helper()
	applicationConfig, err := config.Load()
	if err != nil {
		t.Fatalf("загрузка тестовой конфигурации: %v", err)
	}
	database, err := OpenPostgres(applicationConfig.DatabaseDSN)
	if err != nil {
		t.Fatal(err)
	}
	transaction := database.Begin()
	if transaction.Error != nil {
		t.Fatal(transaction.Error)
	}
	return NewRiscVCommandRepository(transaction), func() {
		if err := transaction.Rollback().Error; err != nil {
			t.Errorf("откат тестовой транзакции: %v", err)
		}
	}
}
