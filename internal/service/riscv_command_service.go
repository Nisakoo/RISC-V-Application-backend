package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/repository"
)

const (
	maximumImageSize = 20 * 1024 * 1024
	maximumVideoSize = 500 * 1024 * 1024
)

var (
	ErrRiscVCommandNotFound = errors.New("команда RISC-V не найдена")
	ErrDraftAlreadyExists   = errors.New("у пользователя уже есть черновик команды RISC-V")
	ErrInvalidRiscVCommand  = errors.New("поля команды RISC-V заполнены некорректно")
	ErrInvalidImage         = errors.New("изображение должно быть PNG или JPEG размером не более 20 МБ")
	ErrInvalidVideo         = errors.New("видео должно быть MP4 размером не более 500 МБ")
)

var mnemonicPattern = regexp.MustCompile(`^[A-Z][A-Z0-9.]{1,15}$`)

type RiscVCommandRepository interface {
	GetByID(ctx context.Context, id int64) (domain.RiscVCommand, error)
	GetDraftByCreatorID(ctx context.Context, creatorID int64) (domain.RiscVCommand, error)
	GetVisible(ctx context.Context, minimumOperandCount int, maximumOperandCount int) ([]domain.RiscVCommand, error)
	GetNextVisible(ctx context.Context, id int64) (domain.RiscVCommand, error)
	CreateDraft(ctx context.Context, creatorID int64, input domain.CreateRiscVCommandDraft, imageKey string, videoKey string) (domain.RiscVCommand, error)
	PublishDraft(ctx context.Context, id int64, creatorID int64, input domain.PublishRiscVCommandDraft) (domain.RiscVCommand, error)
	SoftDelete(ctx context.Context, id int64) error
}

type MediaStorage interface {
	Put(ctx context.Context, directory string, extension string, contentType string, reader io.Reader, size int64) (string, error)
	Remove(ctx context.Context, key string) error
}

type RiscVCommandService struct {
	repository RiscVCommandRepository
	media      MediaStorage
}

func NewRiscVCommandService(repository RiscVCommandRepository, media MediaStorage) *RiscVCommandService {
	return &RiscVCommandService{repository: repository, media: media}
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

	image, imageExtension, err := validatedMedia(input.Image, maximumImageSize, map[string]string{
		"image/png": ".png", "image/jpeg": ".jpg",
	})
	if err != nil {
		return domain.RiscVCommand{}, ErrInvalidImage
	}
	video, videoExtension, err := validatedMedia(input.Video, maximumVideoSize, map[string]string{
		"video/mp4": ".mp4", "video/quicktime": ".mov",
	})
	if err != nil {
		return domain.RiscVCommand{}, ErrInvalidVideo
	}

	imageKey, err := service.media.Put(ctx, "images", imageExtension, image.ContentType, image.Reader, image.Size)
	if err != nil {
		return domain.RiscVCommand{}, err
	}
	videoKey, err := service.media.Put(ctx, "video", videoExtension, video.ContentType, video.Reader, video.Size)
	if err != nil {
		_ = service.media.Remove(ctx, imageKey)
		return domain.RiscVCommand{}, err
	}

	command, err := service.repository.CreateDraft(ctx, creatorID, input, imageKey, videoKey)
	if err != nil {
		_ = service.media.Remove(ctx, imageKey)
		_ = service.media.Remove(ctx, videoKey)
		if errors.Is(err, repository.ErrDraftAlreadyExists) {
			return domain.RiscVCommand{}, ErrDraftAlreadyExists
		}
		return domain.RiscVCommand{}, err
	}
	return command, nil
}

func (service *RiscVCommandService) PublishDraft(ctx context.Context, creatorID int64, input domain.PublishRiscVCommandDraft) (domain.RiscVCommand, error) {
	input.AssemblyExample = strings.TrimSpace(input.AssemblyExample)
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
	if len(input.AssemblyExample) < 3 || len(input.AssemblyExample) > 128 {
		return fmt.Errorf("%w: пример ассемблерной записи должен содержать от 3 до 128 символов", ErrInvalidRiscVCommand)
	}
	if len(input.Description) < 10 || len(input.Description) > 1000 {
		return fmt.Errorf("%w: описание должно содержать от 10 до 1000 символов", ErrInvalidRiscVCommand)
	}
	validOperands := map[domain.InstructionFormat][]int{
		domain.InstructionFormatR: {3},
		domain.InstructionFormatI: {2, 3},
		domain.InstructionFormatS: {2, 3},
		domain.InstructionFormatB: {3},
		domain.InstructionFormatU: {2},
		domain.InstructionFormatJ: {2},
	}
	allowedOperands, formatExists := validOperands[input.Format]
	if !formatExists || !containsInt(allowedOperands, input.OperandCount) {
		return fmt.Errorf("%w: количество операндов не соответствует формату инструкции", ErrInvalidRiscVCommand)
	}
	if !containsInt([]int{16, 32, 48, 64}, input.LengthBits) {
		return fmt.Errorf("%w: длина команды должна равняться 16, 32, 48 или 64 битам", ErrInvalidRiscVCommand)
	}
	return nil
}

func validatedMedia(media domain.UploadedMedia, maximumSize int64, extensions map[string]string) (domain.UploadedMedia, string, error) {
	if media.Reader == nil || media.Size <= 0 || media.Size > maximumSize {
		return domain.UploadedMedia{}, "", errors.New("некорректный размер медиафайла")
	}
	header := make([]byte, 512)
	readBytes, err := io.ReadFull(media.Reader, header)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return domain.UploadedMedia{}, "", err
	}
	detectedContentType := http.DetectContentType(header[:readBytes])
	extension, allowed := extensions[detectedContentType]
	if !allowed {
		return domain.UploadedMedia{}, "", errors.New("недопустимый MIME-тип")
	}
	media.ContentType = detectedContentType
	media.Reader = io.MultiReader(bytes.NewReader(header[:readBytes]), media.Reader)
	return media, extension, nil
}

func containsInt(values []int, expected int) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
