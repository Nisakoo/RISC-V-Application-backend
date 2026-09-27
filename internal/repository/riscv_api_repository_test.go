package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"riscv-lab/internal/domain"
)

func TestRiscVAPIRepositoryLifecycleAndOwnership(t *testing.T) {
	repository, rollback := testRepository(t)
	defer rollback()
	ctx := context.Background()

	ownerLogin := fmt.Sprintf("riscv_api_owner_%d", time.Now().UnixNano())
	ownerID, err := repository.RegisterUser(ctx, ownerLogin, "test-hash")
	if err != nil {
		t.Fatal(err)
	}
	draft, err := repository.CreateDraft(ctx, ownerID, domain.CreateRiscVCommandDraft{
		Mnemonic: "XORI", ImageKey: "images/test.png", VideoKey: "video/test.mp4",
	})
	if err != nil {
		t.Fatal(err)
	}
	if draft.ImageKey != "images/test.png" || draft.VideoKey != "video/test.mp4" || draft.CreatorLogin != ownerLogin {
		t.Fatalf("черновик содержит неверные поля: %+v", draft)
	}
	published, err := repository.PublishDraft(ctx, draft.ID, ownerID, domain.PublishRiscVCommandDraft{
		Description:  "Выполняет побитовое исключающее ИЛИ с immediate.",
		OperandCount: 3, LengthBits: 32,
	})
	if err != nil {
		t.Fatal(err)
	}
	if published.Status != domain.RiscVCommandPublished {
		t.Fatalf("команда не опубликована: %+v", published)
	}

	for attempt := 0; attempt < 2; attempt++ {
		count, err := repository.SetLike(ctx, published.ID, ownerID, true)
		if err != nil || count != 1 {
			t.Fatalf("повторный лайк: count=%d err=%v", count, err)
		}
	}
	count, err := repository.SetLike(ctx, published.ID, ownerID, false)
	if err != nil || count != 0 {
		t.Fatalf("отмена лайка: count=%d err=%v", count, err)
	}
	if err := repository.SoftDeleteOwned(ctx, published.ID, 1); !errors.Is(err, ErrRiscVCommandNotFound) {
		t.Fatalf("чужая команда была удалена: %v", err)
	}
	if err := repository.SoftDeleteOwned(ctx, published.ID, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.SetLike(ctx, published.ID, ownerID, true); !errors.Is(err, ErrRiscVCommandNotFound) {
		t.Fatalf("удалённая команда приняла лайк: %v", err)
	}
	commands, err := repository.GetVisible(ctx, 0, 3)
	if err != nil {
		t.Fatal(err)
	}
	if hasCommandID(commands, published.ID) {
		t.Fatal("удалённая команда попала в список")
	}
	if _, err := repository.RegisterUser(ctx, ownerLogin, "another-test-hash"); !errors.Is(err, ErrLoginAlreadyExists) {
		t.Fatalf("повторный логин: %v", err)
	}
}
