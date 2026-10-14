package service

import (
	"context"
	"errors"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/repository"
)

type RiscVAPIRepository interface {
	RiscVCommandRepository
	SoftDeleteOwned(ctx context.Context, id, creatorID int64) error
	SetLike(ctx context.Context, commandID, userID int64, liked bool) (int64, error)
	IsLiked(ctx context.Context, commandID, userID int64) (bool, error)
}

type RiscVAPIService struct {
	repository RiscVAPIRepository
	commands   *RiscVCommandService
}

func NewRiscVAPIService(repo RiscVAPIRepository) *RiscVAPIService {
	return &RiscVAPIService{repository: repo, commands: NewRiscVCommandService(repo)}
}

func (service *RiscVAPIService) List(ctx context.Context, minimum, maximum int) ([]domain.RiscVCommand, error) {
	commands, _, _, err := service.commands.GetVisibleByOperandRange(ctx, minimum, maximum)
	return commands, err
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
