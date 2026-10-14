package service

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/repository"
)

type fakeAPIRepository struct {
	*fakeRiscVCommandRepository
	ownerID      int64
	deletedID    int64
	registered   string
	passwordHash string
}

func (fake *fakeAPIRepository) SoftDeleteOwned(_ context.Context, id, ownerID int64) error {
	if ownerID != fake.ownerID {
		return repository.ErrRiscVCommandNotFound
	}
	fake.deletedID = id
	return nil
}

func (fake *fakeAPIRepository) SetLike(_ context.Context, _ int64, _ int64, liked bool) (int64, error) {
	if liked {
		return 1, nil
	}
	return 0, nil
}

func (fake *fakeAPIRepository) IsLiked(_ context.Context, _, _ int64) (bool, error) {
	return false, nil
}

func (fake *fakeAPIRepository) RegisterUser(_ context.Context, login, passwordHash string) (int64, error) {
	fake.registered = login
	fake.passwordHash = passwordHash
	return 7, nil
}

func TestRiscVAPIServiceRejectsForeignDraftAndDelete(t *testing.T) {
	fake := &fakeAPIRepository{
		fakeRiscVCommandRepository: &fakeRiscVCommandRepository{draft: &domain.RiscVCommand{ID: 5, CreatorID: 2, Status: domain.RiscVCommandDraft}},
		ownerID:                    2,
	}
	api := NewRiscVAPIService(fake)
	if _, err := api.Publish(context.Background(), 5, domain.PublishRiscVCommandDraft{
		Description: "Описание команды RISC-V достаточной длины.", OperandCount: 3, LengthBits: 32,
	}); !errors.Is(err, ErrRiscVCommandNotFound) {
		t.Fatalf("чужой черновик опубликован: %v", err)
	}
	if err := api.Delete(context.Background(), 5); !errors.Is(err, ErrRiscVCommandNotFound) {
		t.Fatalf("чужая команда удалена: %v", err)
	}
}

func TestRiscVAPIServiceHashesRegistrationPassword(t *testing.T) {
	fake := &fakeAPIRepository{fakeRiscVCommandRepository: &fakeRiscVCommandRepository{}}
	api := NewRiscVAPIService(fake)
	if _, err := api.Register(context.Background(), "rv32_author", "example-password-for-test"); err != nil {
		t.Fatal(err)
	}
	if fake.registered != "rv32_author" || fake.passwordHash == "example-password-for-test" {
		t.Fatal("логин или хеш пароля сохранён некорректно")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(fake.passwordHash), []byte("example-password-for-test")); err != nil {
		t.Fatalf("хеш пароля не соответствует входным данным: %v", err)
	}
	if _, err := api.Register(context.Background(), "x", "short"); !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("неверная регистрация принята: %v", err)
	}
}
