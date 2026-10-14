package service

import (
	"context"
	"errors"
	"testing"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/repository"
)

type fakeAPIRepository struct {
	*fakeRiscVCommandRepository
	ownerID   int64
	deletedID int64
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

func TestRiscVAPIServiceRejectsForeignDraftAndDelete(t *testing.T) {
	fake := &fakeAPIRepository{
		fakeRiscVCommandRepository: &fakeRiscVCommandRepository{draft: &domain.RiscVCommand{ID: 5, CreatorID: 2, Status: domain.RiscVCommandDraft}},
		ownerID:                    2,
	}
	api := NewRiscVAPIService(fake)
	const otherUserID int64 = 1
	if _, err := api.Publish(context.Background(), otherUserID, 5, domain.PublishRiscVCommandDraft{
		Description: "Описание команды RISC-V достаточной длины.", OperandCount: 3, LengthBits: 32,
	}); !errors.Is(err, ErrRiscVCommandNotFound) {
		t.Fatalf("чужой черновик опубликован: %v", err)
	}
	if err := api.Delete(context.Background(), otherUserID, 5); !errors.Is(err, ErrRiscVCommandNotFound) {
		t.Fatalf("чужая команда удалена: %v", err)
	}
}
