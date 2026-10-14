package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/repository"
)

var (
	ErrInvalidUser        = errors.New("логин или пароль заполнены некорректно")
	ErrLoginAlreadyExists = errors.New("пользователь с таким логином уже существует")
)

var loginPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{2,63}$`)

const labThreeUserID int64 = 1

var currentLabThreeUser = sync.OnceValue(func() int64 { return labThreeUserID })

// CurrentUserID возвращает фиксированного пользователя
func CurrentUserID() int64 { return currentLabThreeUser() }

type RiscVAPIRepository interface {
	RiscVCommandRepository
	SoftDeleteOwned(ctx context.Context, id, creatorID int64) error
	SetLike(ctx context.Context, commandID, userID int64, liked bool) (int64, error)
	IsLiked(ctx context.Context, commandID, userID int64) (bool, error)
	RegisterUser(ctx context.Context, login, passwordHash string) (int64, error)
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

// Feed возвращает команду ленты, следующую команду и признак лайка текущего пользователя
func (service *RiscVAPIService) Feed(ctx context.Context, id *int64) (domain.RiscVCommand, domain.RiscVCommand, bool, error) {
	command, following, err := service.commands.GetFeedRiscVCommand(ctx, id)
	if err != nil {
		return domain.RiscVCommand{}, domain.RiscVCommand{}, false, err
	}
	liked, err := service.repository.IsLiked(ctx, command.ID, CurrentUserID())
	if err != nil {
		return domain.RiscVCommand{}, domain.RiscVCommand{}, false, err
	}
	return command, following, liked, nil
}

func (service *RiscVAPIService) Draft(ctx context.Context) (domain.RiscVCommand, error) {
	return service.commands.GetDraft(ctx, CurrentUserID())
}

func (service *RiscVAPIService) Create(ctx context.Context, mnemonic, imageKey, videoKey string) (domain.RiscVCommand, error) {
	if imageKey == "" || videoKey == "" {
		return domain.RiscVCommand{}, ErrInvalidRiscVCommand
	}
	return service.commands.CreateDraft(ctx, CurrentUserID(), domain.CreateRiscVCommandDraft{
		Mnemonic: mnemonic, ImageKey: imageKey, VideoKey: videoKey,
	})
}

func (service *RiscVAPIService) Publish(ctx context.Context, id int64, input domain.PublishRiscVCommandDraft) (domain.RiscVCommand, error) {
	if id <= 0 {
		return domain.RiscVCommand{}, ErrRiscVCommandNotFound
	}
	draft, err := service.Draft(ctx)
	if err != nil {
		return domain.RiscVCommand{}, err
	}
	if draft.ID != id || draft.CreatorID != CurrentUserID() {
		return domain.RiscVCommand{}, ErrRiscVCommandNotFound
	}
	return service.commands.PublishDraft(ctx, CurrentUserID(), input)
}

func (service *RiscVAPIService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrRiscVCommandNotFound
	}
	if err := service.repository.SoftDeleteOwned(ctx, id, CurrentUserID()); err != nil {
		if errors.Is(err, repository.ErrRiscVCommandNotFound) {
			return ErrRiscVCommandNotFound
		}
		return err
	}
	return nil
}

func (service *RiscVAPIService) Like(ctx context.Context, id int64, liked bool) (int64, error) {
	if id <= 0 {
		return 0, ErrRiscVCommandNotFound
	}
	count, err := service.repository.SetLike(ctx, id, CurrentUserID(), liked)
	if errors.Is(err, repository.ErrRiscVCommandNotFound) {
		return 0, ErrRiscVCommandNotFound
	}
	return count, err
}

func (service *RiscVAPIService) Register(ctx context.Context, login, password string) (int64, error) {
	login = strings.TrimSpace(login)
	if !loginPattern.MatchString(login) || len(password) < 8 || len(password) > 72 {
		return 0, ErrInvalidUser
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	id, err := service.repository.RegisterUser(ctx, login, string(hash))
	if errors.Is(err, repository.ErrLoginAlreadyExists) {
		return 0, ErrLoginAlreadyExists
	}
	return id, err
}
