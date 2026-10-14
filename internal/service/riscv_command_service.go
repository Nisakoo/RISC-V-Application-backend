package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/repository"
)

var (
	ErrRiscVCommandNotFound = errors.New("команда RISC-V не найдена")
	ErrDraftAlreadyExists   = errors.New("у пользователя уже есть черновик команды RISC-V")
	ErrInvalidRiscVCommand  = errors.New("поля команды RISC-V заполнены некорректно")
)

var mnemonicPattern = regexp.MustCompile(`^[A-Z][A-Z0-9.]{1,15}$`)

type RiscVCommandRepository interface {
	GetByID(ctx context.Context, id int64) (domain.RiscVCommand, error)
	GetDraftByCreatorID(ctx context.Context, creatorID int64) (domain.RiscVCommand, error)
	GetVisible(ctx context.Context, minimumOperandCount int, maximumOperandCount int) ([]domain.RiscVCommand, error)
	GetNextVisible(ctx context.Context, id int64) (domain.RiscVCommand, error)
	CreateDraft(ctx context.Context, creatorID int64, input domain.CreateRiscVCommandDraft) (domain.RiscVCommand, error)
	PublishDraft(ctx context.Context, id int64, creatorID int64, input domain.PublishRiscVCommandDraft) (domain.RiscVCommand, error)
	SoftDelete(ctx context.Context, id int64) error
}

type RiscVCommandService struct {
	repository RiscVCommandRepository
}

func NewRiscVCommandService(repository RiscVCommandRepository) *RiscVCommandService {
	return &RiscVCommandService{repository: repository}
}

func (service *RiscVCommandService) GetFeedRiscVCommand(ctx context.Context, id *int64) (domain.RiscVCommand, domain.RiscVCommand, error) {
	var command domain.RiscVCommand
	var err error

	if id == nil {
		commands, repositoryError := service.repository.GetVisible(ctx, 0, 3)
		if repositoryError != nil {
			return domain.RiscVCommand{}, domain.RiscVCommand{}, repositoryError
		}
		if len(commands) == 0 {
			return domain.RiscVCommand{}, domain.RiscVCommand{}, ErrRiscVCommandNotFound
		}
		command = commands[0]
	} else {
		command, err = service.repository.GetByID(ctx, *id)
		if err != nil {
			if errors.Is(err, repository.ErrRiscVCommandNotFound) {
				return domain.RiscVCommand{}, domain.RiscVCommand{}, ErrRiscVCommandNotFound
			}
			return domain.RiscVCommand{}, domain.RiscVCommand{}, err
		}
		if command.Status != domain.RiscVCommandPublished {
			return domain.RiscVCommand{}, domain.RiscVCommand{}, ErrRiscVCommandNotFound
		}
	}

	nextCommand, err := service.repository.GetNextVisible(ctx, command.ID)
	if errors.Is(err, repository.ErrRiscVCommandNotFound) {
		return domain.RiscVCommand{}, domain.RiscVCommand{}, ErrRiscVCommandNotFound
	}
	if err != nil {
		return domain.RiscVCommand{}, domain.RiscVCommand{}, err
	}
	return command, nextCommand, nil
}

func (service *RiscVCommandService) GetDraft(ctx context.Context, creatorID int64) (domain.RiscVCommand, error) {
	command, err := service.repository.GetDraftByCreatorID(ctx, creatorID)
	if errors.Is(err, repository.ErrRiscVCommandNotFound) {
		return domain.RiscVCommand{}, ErrRiscVCommandNotFound
	}
	if err != nil {
		return domain.RiscVCommand{}, err
	}
	return command, nil
}

func (service *RiscVCommandService) GetVisibleByOperandRange(ctx context.Context, minimumOperandCount int, maximumOperandCount int) ([]domain.RiscVCommand, int, int, error) {
	if minimumOperandCount > maximumOperandCount {
		minimumOperandCount, maximumOperandCount = maximumOperandCount, minimumOperandCount
	}
	commands, err := service.repository.GetVisible(ctx, minimumOperandCount, maximumOperandCount)
	return commands, minimumOperandCount, maximumOperandCount, err
}

func (service *RiscVCommandService) CreateDraft(ctx context.Context, creatorID int64, input domain.CreateRiscVCommandDraft) (domain.RiscVCommand, error) {
	input.Mnemonic = strings.ToUpper(strings.TrimSpace(input.Mnemonic))
	if !mnemonicPattern.MatchString(input.Mnemonic) {
		return domain.RiscVCommand{}, fmt.Errorf("%w: мнемоника должна содержать от 2 до 16 латинских букв, цифр или точек", ErrInvalidRiscVCommand)
	}

	// url медиа обязательны. Если файлы не переданы (страница добавления SSR),
	// сохраняются ключи несуществующих объектов: вместо них показываются медиа по умолчанию.
	if input.ImageKey == "" {
		input.ImageKey = "images/missing/" + uuid.NewString() + ".png"
	}
	if input.VideoKey == "" {
		input.VideoKey = "video/missing/" + uuid.NewString() + ".mp4"
	}

	command, err := service.repository.CreateDraft(ctx, creatorID, input)
	if err != nil {
		if errors.Is(err, repository.ErrDraftAlreadyExists) {
			return domain.RiscVCommand{}, ErrDraftAlreadyExists
		}
		return domain.RiscVCommand{}, err
	}
	return command, nil
}

func (service *RiscVCommandService) PublishDraft(ctx context.Context, creatorID int64, input domain.PublishRiscVCommandDraft) (domain.RiscVCommand, error) {
	input.Description = strings.TrimSpace(input.Description)
	if err := validatePublication(input); err != nil {
		return domain.RiscVCommand{}, err
	}

	draft, err := service.GetDraft(ctx, creatorID)
	if err != nil {
		return domain.RiscVCommand{}, err
	}
	command, err := service.repository.PublishDraft(ctx, draft.ID, creatorID, input)
	if errors.Is(err, repository.ErrRiscVCommandNotFound) {
		return domain.RiscVCommand{}, ErrRiscVCommandNotFound
	}
	return command, err
}

func (service *RiscVCommandService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrRiscVCommandNotFound
	}
	if err := service.repository.SoftDelete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrRiscVCommandNotFound) {
			return ErrRiscVCommandNotFound
		}
		return err
	}
	return nil
}

func validatePublication(input domain.PublishRiscVCommandDraft) error {
	if len(input.Description) < 10 || len(input.Description) > 1000 {
		return fmt.Errorf("%w: описание должно содержать от 10 до 1000 символов", ErrInvalidRiscVCommand)
	}
	if input.OperandCount < 0 || input.OperandCount > 3 {
		return fmt.Errorf("%w: количество операндов должно быть от 0 до 3", ErrInvalidRiscVCommand)
	}
	if !containsInt([]int{16, 32, 48, 64}, input.LengthBits) {
		return fmt.Errorf("%w: длина команды должна равняться 16, 32, 48 или 64 битам", ErrInvalidRiscVCommand)
	}
	return nil
}

func containsInt(values []int, expected int) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
