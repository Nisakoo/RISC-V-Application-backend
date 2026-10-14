package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"riscv-lab/internal/domain"
)

var (
	ErrLoginAlreadyExists = errors.New("пользователь с таким логином уже существует")
	ErrUserNotFound       = errors.New("пользователь не найден")
)

type userRecord struct {
	ID           int64  `gorm:"primaryKey;column:id"`
	Login        string `gorm:"column:login"`
	PasswordHash string `gorm:"column:password_hash"`
	Role         string `gorm:"column:role"`
}

func (userRecord) TableName() string { return "users" }

func (repository *RiscVCommandRepository) SoftDeleteOwned(ctx context.Context, id, creatorID int64) error {
	result := repository.db.WithContext(ctx).Model(&riscvCommandRecord{}).
		Where("id = ? AND creator_id = ? AND status IN ?", id, creatorID, []domain.RiscVCommandStatus{domain.RiscVCommandDraft, domain.RiscVCommandPublished}).
		Update("status", domain.RiscVCommandDeleted)
	if result.Error != nil {
		return fmt.Errorf("логическое удаление команды RISC-V: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrRiscVCommandNotFound
	}
	return nil
}

func (repository *RiscVCommandRepository) SetLike(ctx context.Context, commandID, userID int64, liked bool) (int64, error) {
	var count int64
	err := repository.db.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		var command riscvCommandRecord
		if err := transaction.Where("id = ? AND status = ?", commandID, domain.RiscVCommandPublished).Take(&command).Error; err != nil {
			return mapNotFound(err)
		}
		if liked {
			like := riscvCommandLikeRecord{UserID: userID, RiscVCommandID: commandID}
			if err := transaction.Clauses(clause.OnConflict{DoNothing: true}).Create(&like).Error; err != nil {
				return err
			}
		} else {
			if err := transaction.Where("user_id = ? AND riscv_command_id = ?", userID, commandID).Delete(&riscvCommandLikeRecord{}).Error; err != nil {
				return err
			}
		}
		return transaction.Model(&riscvCommandLikeRecord{}).Where("riscv_command_id = ?", commandID).Count(&count).Error
	})
	if err != nil {
		return 0, fmt.Errorf("изменение лайка команды RISC-V: %w", err)
	}
	return count, nil
}

// IsLiked проверяет, поставил ли пользователь лайк команде RISC-V
func (repository *RiscVCommandRepository) IsLiked(ctx context.Context, commandID, userID int64) (bool, error) {
	var liked bool
	err := repository.db.WithContext(ctx).Raw(
		"SELECT EXISTS (SELECT 1 FROM riscv_command_likes WHERE user_id = ? AND riscv_command_id = ?)",
		userID, commandID,
	).Scan(&liked).Error
	if err != nil {
		return false, fmt.Errorf("проверка лайка команды RISC-V: %w", err)
	}
	return liked, nil
}

type riscvCommandLikeRecord struct {
	ID             int64 `gorm:"primaryKey;column:id"`
	UserID         int64 `gorm:"column:user_id"`
	RiscVCommandID int64 `gorm:"column:riscv_command_id"`
}

func (riscvCommandLikeRecord) TableName() string { return "riscv_command_likes" }

// GetUserByLogin возвращает пользователя с хешем пароля для аутентификации
func (repository *RiscVCommandRepository) GetUserByLogin(ctx context.Context, login string) (domain.User, error) {
	var user userRecord
	if err := repository.db.WithContext(ctx).Where("login = ?", login).Take(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("получение пользователя: %w", err)
	}
	return domain.User{ID: user.ID, Login: user.Login, PasswordHash: user.PasswordHash, Role: domain.UserRole(user.Role)}, nil
}

func (repository *RiscVCommandRepository) RegisterUser(ctx context.Context, login, passwordHash string) (int64, error) {
	user := userRecord{Login: login, PasswordHash: passwordHash, Role: string(domain.UserRoleUser)}
	if err := repository.db.WithContext(ctx).Create(&user).Error; err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" {
			return 0, ErrLoginAlreadyExists
		}
		return 0, fmt.Errorf("регистрация пользователя: %w", err)
	}
	return user.ID, nil
}
