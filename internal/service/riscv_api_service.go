package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/repository"
)

type RiscVAPIRepository interface {
	RiscVCommandRepository
	SoftDeleteOwned(ctx context.Context, id, creatorID int64) error
	SetLike(ctx context.Context, commandID, userID int64, liked bool) (int64, error)
	IsLiked(ctx context.Context, commandID, userID int64) (bool, error)
	GetVisiblePage(ctx context.Context, minimum, maximum, limit, offset int) ([]domain.RiscVCommand, int64, error)
	RandomVisibleIDs(ctx context.Context, limit int) ([]int64, error)
	GetVisibleByIDs(ctx context.Context, ids []int64) ([]domain.RiscVCommand, error)
	LikedIDs(ctx context.Context, userID int64, ids []int64) (map[int64]bool, error)
}

// FeedStore хранит порцию ленты сессии (Redis)
type FeedStore interface {
	Load(ctx context.Context, sessionID string) (domain.FeedBatch, bool, error)
	Save(ctx context.Context, batch domain.FeedBatch, ttl time.Duration) error
}

const (
	// FeedBatchSize — размер порции ленты сессии
	FeedBatchSize = 100
	// feedSessionTTL — время жизни порции ленты в Redis
	feedSessionTTL = 30 * time.Minute
)

var (
	ErrInvalidFeedSession = errors.New("некорректный идентификатор сессии ленты")
	ErrFeedUnavailable    = errors.New("лента сессии недоступна")
)

type RiscVAPIService struct {
	repository RiscVAPIRepository
	commands   *RiscVCommandService
	feed       FeedStore
}

// WithFeedStore подключает хранилище лент сессий
func (service *RiscVAPIService) WithFeedStore(store FeedStore) *RiscVAPIService {
	service.feed = store
	return service
}

func NewRiscVAPIService(repo RiscVAPIRepository) *RiscVAPIService {
	return &RiscVAPIService{repository: repo, commands: NewRiscVCommandService(repo)}
}

// List возвращает страницу опубликованных команд и общее количество команд фильтра
func (service *RiscVAPIService) List(ctx context.Context, minimum, maximum, page, limit int) ([]domain.RiscVCommand, int64, error) {
	if minimum > maximum {
		minimum, maximum = maximum, minimum
	}
	return service.repository.GetVisiblePage(ctx, minimum, maximum, limit, (page-1)*limit)
}

// FeedBatch возвращает текущую порцию ленты сессии из Redis.
// Без sessionID создаётся новая сессия; при refresh = true или неизвестной (истёкшей) сессии
// формируется новая порция из FeedBatchSize случайных команд, нумерация продолжается.
func (service *RiscVAPIService) FeedBatch(ctx context.Context, userID int64, sessionID string, refresh bool) (domain.FeedBatch, []domain.FeedBatchItem, error) {
	if service.feed == nil {
		return domain.FeedBatch{}, nil, ErrFeedUnavailable
	}
	if sessionID == "" {
		sessionID = strings.ReplaceAll(uuid.NewString(), "-", "")
	} else if parsed, err := uuid.Parse(sessionID); err != nil || strings.ReplaceAll(parsed.String(), "-", "") != sessionID {
		return domain.FeedBatch{}, nil, ErrInvalidFeedSession
	}

	batch, found, err := service.feed.Load(ctx, sessionID)
	if err != nil {
		return domain.FeedBatch{}, nil, err
	}
	if !found || refresh {
		next := domain.FeedBatch{SessionID: sessionID, BatchNumber: 1, FirstPosition: 1}
		if found {
			next.BatchNumber = batch.BatchNumber + 1
			next.FirstPosition = batch.FirstPosition + len(batch.IDs)
		}
		next.IDs, err = service.repository.RandomVisibleIDs(ctx, FeedBatchSize)
		if err != nil {
			return domain.FeedBatch{}, nil, err
		}
		if len(next.IDs) == 0 {
			return domain.FeedBatch{}, nil, ErrRiscVCommandNotFound
		}
		if err := service.feed.Save(ctx, next, feedSessionTTL); err != nil {
			return domain.FeedBatch{}, nil, err
		}
		batch = next
	}

	commands, err := service.repository.GetVisibleByIDs(ctx, batch.IDs)
	if err != nil {
		return domain.FeedBatch{}, nil, err
	}
	liked, err := service.repository.LikedIDs(ctx, userID, batch.IDs)
	if err != nil {
		return domain.FeedBatch{}, nil, err
	}
	byID := make(map[int64]domain.RiscVCommand, len(commands))
	for _, command := range commands {
		byID[command.ID] = command
	}
	// Порядок и номера берутся из Redis; команды, удалённые после формирования порции, пропускаются
	items := make([]domain.FeedBatchItem, 0, len(batch.IDs))
	for index, id := range batch.IDs {
		if command, exists := byID[id]; exists {
			items = append(items, domain.FeedBatchItem{Command: command, Position: batch.FirstPosition + index, Liked: liked[id]})
		}
	}
	return batch, items, nil
}

// Feed возвращает команду ленты, следующую команду и признак лайка текущего пользователя.
// При next=true возвращается команда, следующая за указанным id.
// Для гостя (userID = 0) признак лайка всегда false.
func (service *RiscVAPIService) Feed(ctx context.Context, userID int64, id *int64, next bool) (domain.RiscVCommand, domain.RiscVCommand, bool, error) {
	command, following, err := service.commands.GetFeedRiscVCommand(ctx, id)
	if err != nil {
		return domain.RiscVCommand{}, domain.RiscVCommand{}, false, err
	}
	if next && id != nil {
		command = following
		following, err = service.repository.GetNextVisible(ctx, command.ID)
		if err != nil {
			return domain.RiscVCommand{}, domain.RiscVCommand{}, false, err
		}
	}
	if userID <= 0 {
		return command, following, false, nil
	}
	liked, err := service.repository.IsLiked(ctx, command.ID, userID)
	if err != nil {
		return domain.RiscVCommand{}, domain.RiscVCommand{}, false, err
	}
	return command, following, liked, nil
}

func (service *RiscVAPIService) Draft(ctx context.Context, userID int64) (domain.RiscVCommand, error) {
	return service.commands.GetDraft(ctx, userID)
}

// Create создаёт черновик, создатель которого определяется по JWT
func (service *RiscVAPIService) Create(ctx context.Context, userID int64, mnemonic, imageKey, videoKey string) (domain.RiscVCommand, error) {
	if imageKey == "" || videoKey == "" {
		return domain.RiscVCommand{}, ErrInvalidRiscVCommand
	}
	return service.commands.CreateDraft(ctx, userID, domain.CreateRiscVCommandDraft{
		Mnemonic: mnemonic, ImageKey: imageKey, VideoKey: videoKey,
	})
}

func (service *RiscVAPIService) Publish(ctx context.Context, userID, id int64, input domain.PublishRiscVCommandDraft) (domain.RiscVCommand, error) {
	if id <= 0 {
		return domain.RiscVCommand{}, ErrRiscVCommandNotFound
	}
	draft, err := service.Draft(ctx, userID)
	if err != nil {
		return domain.RiscVCommand{}, err
	}
	if draft.ID != id || draft.CreatorID != userID {
		return domain.RiscVCommand{}, ErrRiscVCommandNotFound
	}
	return service.commands.PublishDraft(ctx, userID, input)
}

func (service *RiscVAPIService) Delete(ctx context.Context, userID, id int64) error {
	if id <= 0 {
		return ErrRiscVCommandNotFound
	}
	if err := service.repository.SoftDeleteOwned(ctx, id, userID); err != nil {
		if errors.Is(err, repository.ErrRiscVCommandNotFound) {
			return ErrRiscVCommandNotFound
		}
		return err
	}
	return nil
}

func (service *RiscVAPIService) Like(ctx context.Context, userID, id int64, liked bool) (int64, error) {
	if id <= 0 {
		return 0, ErrRiscVCommandNotFound
	}
	count, err := service.repository.SetLike(ctx, id, userID, liked)
	if errors.Is(err, repository.ErrRiscVCommandNotFound) {
		return 0, ErrRiscVCommandNotFound
	}
	return count, err
}
