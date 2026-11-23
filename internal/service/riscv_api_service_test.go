package service

import (
	"context"
	"errors"
	"testing"
	"time"

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

func (fake *fakeAPIRepository) GetVisiblePage(ctx context.Context, minimum, maximum, limit, offset int) ([]domain.RiscVCommand, int64, error) {
	visible, _ := fake.GetVisible(ctx, minimum, maximum)
	total := int64(len(visible))
	if offset >= len(visible) {
		return nil, total, nil
	}
	return visible[offset:min(offset+limit, len(visible))], total, nil
}

func (fake *fakeAPIRepository) RandomVisibleIDs(ctx context.Context, limit int) ([]int64, error) {
	visible, _ := fake.GetVisible(ctx, 0, 3)
	ids := make([]int64, 0, len(visible))
	for _, command := range visible {
		ids = append(ids, command.ID)
	}
	return ids[:min(limit, len(ids))], nil
}

func (fake *fakeAPIRepository) GetVisibleByIDs(_ context.Context, ids []int64) ([]domain.RiscVCommand, error) {
	result := make([]domain.RiscVCommand, 0, len(ids))
	for _, command := range fake.commands {
		for _, id := range ids {
			if command.ID == id && command.Status == domain.RiscVCommandPublished {
				result = append(result, command)
			}
		}
	}
	return result, nil
}

func (fake *fakeAPIRepository) LikedIDs(_ context.Context, _ int64, _ []int64) (map[int64]bool, error) {
	return map[int64]bool{}, nil
}

// memoryFeedStore хранит порции лент в памяти для тестов без Redis
type memoryFeedStore map[string]domain.FeedBatch

func (store memoryFeedStore) Load(_ context.Context, sessionID string) (domain.FeedBatch, bool, error) {
	batch, found := store[sessionID]
	return batch, found, nil
}

func (store memoryFeedStore) Save(_ context.Context, batch domain.FeedBatch, _ time.Duration) error {
	store[batch.SessionID] = batch
	return nil
}

func TestRiscVAPIServiceListPaginates(t *testing.T) {
	commands := make([]domain.RiscVCommand, 0)
	for id := int64(1); id <= 5; id++ {
		commands = append(commands, domain.RiscVCommand{ID: id, Status: domain.RiscVCommandPublished, OperandCount: 3})
	}
	api := NewRiscVAPIService(&fakeAPIRepository{fakeRiscVCommandRepository: &fakeRiscVCommandRepository{commands: commands}})
	page, total, err := api.List(context.Background(), 3, 0, 2, 2)
	if err != nil || total != 5 || len(page) != 2 || page[0].ID != 3 {
		t.Fatalf("страница %v, всего %d, ошибка %v", page, total, err)
	}
}

func TestRiscVAPIServiceFeedBatchContinuesNumbering(t *testing.T) {
	commands := []domain.RiscVCommand{
		{ID: 1, Status: domain.RiscVCommandPublished}, {ID: 2, Status: domain.RiscVCommandPublished}, {ID: 3, Status: domain.RiscVCommandDeleted},
	}
	api := NewRiscVAPIService(&fakeAPIRepository{fakeRiscVCommandRepository: &fakeRiscVCommandRepository{commands: commands}}).WithFeedStore(memoryFeedStore{})

	first, items, err := api.FeedBatch(context.Background(), 0, "", false)
	if err != nil || first.SessionID == "" || first.FirstPosition != 1 || len(items) != 2 || items[1].Position != 2 {
		t.Fatalf("первая порция %+v, %v, ошибка %v", first, items, err)
	}
	// Повторный запрос без refresh возвращает ту же порцию из хранилища
	again, _, err := api.FeedBatch(context.Background(), 0, first.SessionID, false)
	if err != nil || again.BatchNumber != 1 {
		t.Fatalf("повторная порция %+v, ошибка %v", again, err)
	}
	second, items, err := api.FeedBatch(context.Background(), 0, first.SessionID, true)
	if err != nil || second.BatchNumber != 2 || second.FirstPosition != 3 || items[0].Position != 3 {
		t.Fatalf("вторая порция %+v, %v, ошибка %v", second, items, err)
	}
	if _, _, err := api.FeedBatch(context.Background(), 0, "not-a-session", false); !errors.Is(err, ErrInvalidFeedSession) {
		t.Fatalf("некорректная сессия принята: %v", err)
	}
}
