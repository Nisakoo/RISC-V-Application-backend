package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"riscv-lab/internal/domain"
)

var (
	ErrRiscVCommandNotFound = errors.New("risc-v command not found")
	ErrDraftAlreadyExists   = errors.New("risc-v command draft already exists")
)

type riscvCommandRecord struct {
	ID           int64                     `gorm:"primaryKey;column:id"`
	Mnemonic     string                    `gorm:"column:mnemonic"`
	Description  string                    `gorm:"column:description"`
	OperandCount int                       `gorm:"column:operand_count"`
	LengthBits   int                       `gorm:"column:length_bits"`
	ImageKey     string                    `gorm:"column:image_key"`
	VideoKey     string                    `gorm:"column:video_key"`
	Status       domain.RiscVCommandStatus `gorm:"column:status"`
	CreatedAt    time.Time                 `gorm:"column:created_at"`
	PublishedAt  *time.Time                `gorm:"column:published_at"`
	CreatorID    int64                     `gorm:"column:creator_id"`
	CreatorLogin string                    `gorm:"column:creator_login;->"`
	LikeCount    int64                     `gorm:"column:like_count;->"`
}

func (riscvCommandRecord) TableName() string {
	return "riscv_commands"
}

type RiscVCommandRepository struct {
	db *gorm.DB
}

func OpenPostgres(dsn string) (*gorm.DB, error) {
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("подключение к PostgreSQL: %w", err)
	}
	return database, nil
}

func NewRiscVCommandRepository(database *gorm.DB) *RiscVCommandRepository {
	return &RiscVCommandRepository{db: database}
}

func (repository *RiscVCommandRepository) GetByID(ctx context.Context, id int64) (domain.RiscVCommand, error) {
	var record riscvCommandRecord
	result := repository.withLikes(ctx).
		Where("riscv_commands.id = ?", id).
		Take(&record)
	if result.Error != nil {
		return domain.RiscVCommand{}, mapNotFound(result.Error)
	}
	return record.toDomain(), nil
}

func (repository *RiscVCommandRepository) GetDraftByCreatorID(ctx context.Context, creatorID int64) (domain.RiscVCommand, error) {
	var record riscvCommandRecord
	result := repository.withLikes(ctx).
		Where("riscv_commands.creator_id = ? AND riscv_commands.status = ?", creatorID, domain.RiscVCommandDraft).
		Order("riscv_commands.id").
		Take(&record)
	if result.Error != nil {
		return domain.RiscVCommand{}, mapNotFound(result.Error)
	}
	return record.toDomain(), nil
}

func (repository *RiscVCommandRepository) GetVisible(ctx context.Context, minimumOperandCount int, maximumOperandCount int) ([]domain.RiscVCommand, error) {
	var records []riscvCommandRecord
	result := repository.withLikes(ctx).
		Where("riscv_commands.status = ?", domain.RiscVCommandPublished).
		Where("riscv_commands.operand_count BETWEEN ? AND ?", minimumOperandCount, maximumOperandCount).
		Order("riscv_commands.id").
		Find(&records)
	if result.Error != nil {
		return nil, fmt.Errorf("получение команд RISC-V: %w", result.Error)
	}
	return recordsToDomain(records), nil
}

// GetFirstVisible возвращает первую опубликованную команду ленты без загрузки всего каталога
func (repository *RiscVCommandRepository) GetFirstVisible(ctx context.Context) (domain.RiscVCommand, error) {
	var record riscvCommandRecord
	result := repository.withLikes(ctx).
		Where("riscv_commands.status = ?", domain.RiscVCommandPublished).
		Order("riscv_commands.id").
		Take(&record)
	if result.Error != nil {
		return domain.RiscVCommand{}, mapNotFound(result.Error)
	}
	return record.toDomain(), nil
}

// GetVisiblePage возвращает страницу опубликованных команд и общее количество команд фильтра.
// Страница выбирается по идентификаторам в подзапросе, чтобы агрегация лайков выполнялась только
// для limit строк, а не для всего каталога.
func (repository *RiscVCommandRepository) GetVisiblePage(ctx context.Context, minimumOperandCount, maximumOperandCount, limit, offset int) ([]domain.RiscVCommand, int64, error) {
	filter := func(query *gorm.DB) *gorm.DB {
		return query.
			Where("riscv_commands.status = ?", domain.RiscVCommandPublished).
			Where("riscv_commands.operand_count BETWEEN ? AND ?", minimumOperandCount, maximumOperandCount)
	}

	var total int64
	if err := filter(repository.db.WithContext(ctx).Model(&riscvCommandRecord{})).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("подсчёт команд RISC-V: %w", err)
	}

	pageIDs := filter(repository.db.WithContext(ctx).Model(&riscvCommandRecord{})).
		Select("riscv_commands.id").
		Order("riscv_commands.id").
		Limit(limit).
		Offset(offset)

	var records []riscvCommandRecord
	result := repository.withLikes(ctx).
		Where("riscv_commands.id IN (?)", pageIDs).
		Order("riscv_commands.id").
		Find(&records)
	if result.Error != nil {
		return nil, 0, fmt.Errorf("получение страницы команд RISC-V: %w", result.Error)
	}
	return recordsToDomain(records), total, nil
}

// RandomVisibleIDs возвращает limit случайных неповторяющихся идентификаторов опубликованных команд
func (repository *RiscVCommandRepository) RandomVisibleIDs(ctx context.Context, limit int) ([]int64, error) {
	var ids []int64
	err := repository.db.WithContext(ctx).Raw(
		"SELECT id FROM riscv_commands WHERE status = ? ORDER BY random() LIMIT ?",
		domain.RiscVCommandPublished, limit,
	).Scan(&ids).Error
	if err != nil {
		return nil, fmt.Errorf("случайные команды RISC-V для ленты: %w", err)
	}
	return ids, nil
}

// GetVisibleByIDs возвращает опубликованные команды из списка идентификаторов (порядок не гарантируется)
func (repository *RiscVCommandRepository) GetVisibleByIDs(ctx context.Context, ids []int64) ([]domain.RiscVCommand, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var records []riscvCommandRecord
	result := repository.withLikes(ctx).
		Where("riscv_commands.status = ? AND riscv_commands.id IN ?", domain.RiscVCommandPublished, ids).
		Find(&records)
	if result.Error != nil {
		return nil, fmt.Errorf("получение команд RISC-V по идентификаторам: %w", result.Error)
	}
	return recordsToDomain(records), nil
}

// LikedIDs возвращает множество команд из списка, которым пользователь поставил лайк
func (repository *RiscVCommandRepository) LikedIDs(ctx context.Context, userID int64, ids []int64) (map[int64]bool, error) {
	liked := make(map[int64]bool)
	if userID <= 0 || len(ids) == 0 {
		return liked, nil
	}
	var found []int64
	err := repository.db.WithContext(ctx).Raw(
		"SELECT riscv_command_id FROM riscv_command_likes WHERE user_id = ? AND riscv_command_id IN ?",
		userID, ids,
	).Scan(&found).Error
	if err != nil {
		return nil, fmt.Errorf("лайки команд RISC-V: %w", err)
	}
	for _, id := range found {
		liked[id] = true
	}
	return liked, nil
}

func (repository *RiscVCommandRepository) GetNextVisible(ctx context.Context, id int64) (domain.RiscVCommand, error) {
	current, err := repository.GetByID(ctx, id)
	if err != nil {
		return domain.RiscVCommand{}, err
	}
	if current.Status != domain.RiscVCommandPublished {
		return domain.RiscVCommand{}, ErrRiscVCommandNotFound
	}

	var record riscvCommandRecord
	result := repository.withLikes(ctx).
		Where("riscv_commands.status = ? AND riscv_commands.id > ?", domain.RiscVCommandPublished, id).
		Order("riscv_commands.id").
		Take(&record)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		result = repository.withLikes(ctx).
			Where("riscv_commands.status = ?", domain.RiscVCommandPublished).
			Order("riscv_commands.id").
			Take(&record)
	}
	if result.Error != nil {
		return domain.RiscVCommand{}, mapNotFound(result.Error)
	}
	return record.toDomain(), nil
}

func (repository *RiscVCommandRepository) CreateDraft(ctx context.Context, creatorID int64, input domain.CreateRiscVCommandDraft) (domain.RiscVCommand, error) {
	record := riscvCommandRecord{
		Mnemonic:  input.Mnemonic,
		ImageKey:  input.ImageKey,
		VideoKey:  input.VideoKey,
		Status:    domain.RiscVCommandDraft,
		CreatorID: creatorID,
		CreatedAt: time.Now().UTC(),
	}
	query := repository.db.WithContext(ctx).Omit("Description", "OperandCount", "LengthBits")
	if err := query.Create(&record).Error; err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" {
			return domain.RiscVCommand{}, ErrDraftAlreadyExists
		}
		return domain.RiscVCommand{}, fmt.Errorf("создание черновика команды RISC-V: %w", err)
	}
	return repository.GetByID(ctx, record.ID)
}

func (repository *RiscVCommandRepository) PublishDraft(ctx context.Context, id int64, creatorID int64, input domain.PublishRiscVCommandDraft) (domain.RiscVCommand, error) {
	now := time.Now().UTC()
	result := repository.db.WithContext(ctx).
		Model(&riscvCommandRecord{}).
		Where("id = ? AND creator_id = ? AND status = ?", id, creatorID, domain.RiscVCommandDraft).
		Updates(map[string]any{
			"description":   input.Description,
			"operand_count": input.OperandCount,
			"length_bits":   input.LengthBits,
			"status":        domain.RiscVCommandPublished,
			"published_at":  now,
		})
	if result.Error != nil {
		return domain.RiscVCommand{}, fmt.Errorf("публикация команды RISC-V: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.RiscVCommand{}, ErrRiscVCommandNotFound
	}
	return repository.GetByID(ctx, id)
}

func (repository *RiscVCommandRepository) SoftDelete(ctx context.Context, id int64) error {
	result := repository.db.WithContext(ctx).Exec(
		"UPDATE riscv_commands SET status = ? WHERE id = ? AND status <> ?",
		domain.RiscVCommandDeleted,
		id,
		domain.RiscVCommandDeleted,
	)
	if result.Error != nil {
		return fmt.Errorf("логическое удаление команды RISC-V: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrRiscVCommandNotFound
	}
	return nil
}

func (repository *RiscVCommandRepository) withLikes(ctx context.Context) *gorm.DB {
	return repository.db.WithContext(ctx).
		Model(&riscvCommandRecord{}).
		Select("riscv_commands.*, users.login AS creator_login, COUNT(riscv_command_likes.id) AS like_count").
		Joins("JOIN users ON users.id = riscv_commands.creator_id").
		Joins("LEFT JOIN riscv_command_likes ON riscv_command_likes.riscv_command_id = riscv_commands.id").
		Group("riscv_commands.id, users.id")
}

func (record riscvCommandRecord) toDomain() domain.RiscVCommand {
	return domain.RiscVCommand{
		ID:           record.ID,
		Mnemonic:     record.Mnemonic,
		Description:  record.Description,
		OperandCount: record.OperandCount,
		LengthBits:   record.LengthBits,
		ImageKey:     record.ImageKey,
		VideoKey:     record.VideoKey,
		Status:       record.Status,
		CreatedAt:    record.CreatedAt,
		PublishedAt:  record.PublishedAt,
		CreatorID:    record.CreatorID,
		CreatorLogin: record.CreatorLogin,
		LikeCount:    record.LikeCount,
	}
}

func recordsToDomain(records []riscvCommandRecord) []domain.RiscVCommand {
	commands := make([]domain.RiscVCommand, 0, len(records))
	for _, record := range records {
		commands = append(commands, record.toDomain())
	}
	return commands
}

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrRiscVCommandNotFound
	}
	return err
}
