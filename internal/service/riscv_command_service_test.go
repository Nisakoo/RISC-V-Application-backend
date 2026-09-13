package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/repository"
)

type fakeRiscVCommandRepository struct {
	commands []domain.RiscVCommand
	draft    *domain.RiscVCommand
	deleted  int64
}

func (fake *fakeRiscVCommandRepository) GetByID(_ context.Context, id int64) (domain.RiscVCommand, error) {
	for _, command := range fake.commands {
		if command.ID == id {
			return command, nil
		}
	}
	return domain.RiscVCommand{}, repository.ErrRiscVCommandNotFound
}

func (fake *fakeRiscVCommandRepository) GetDraftByCreatorID(_ context.Context, _ int64) (domain.RiscVCommand, error) {
	if fake.draft == nil {
		return domain.RiscVCommand{}, repository.ErrRiscVCommandNotFound
	}
	return *fake.draft, nil
}

func (fake *fakeRiscVCommandRepository) GetVisible(_ context.Context, minimum int, maximum int) ([]domain.RiscVCommand, error) {
	commands := make([]domain.RiscVCommand, 0)
	for _, command := range fake.commands {
		if command.Status == domain.RiscVCommandPublished && command.OperandCount >= minimum && command.OperandCount <= maximum {
			commands = append(commands, command)
		}
	}
	return commands, nil
}

func (fake *fakeRiscVCommandRepository) GetNextVisible(_ context.Context, id int64) (domain.RiscVCommand, error) {
	for index, command := range fake.commands {
		if command.ID == id {
			for offset := 1; offset <= len(fake.commands); offset++ {
				next := fake.commands[(index+offset)%len(fake.commands)]
				if next.Status == domain.RiscVCommandPublished {
					return next, nil
				}
			}
		}
	}
	return domain.RiscVCommand{}, repository.ErrRiscVCommandNotFound
}

func (fake *fakeRiscVCommandRepository) CreateDraft(_ context.Context, creatorID int64, input domain.CreateRiscVCommandDraft, imageKey string, videoKey string) (domain.RiscVCommand, error) {
	if fake.draft != nil {
		return domain.RiscVCommand{}, repository.ErrDraftAlreadyExists
	}
	command := domain.RiscVCommand{ID: 101, Mnemonic: input.Mnemonic, ImageKey: imageKey, VideoKey: videoKey, Status: domain.RiscVCommandDraft, CreatorID: creatorID}
	fake.draft = &command
	fake.commands = append(fake.commands, command)
	return command, nil
}

func (fake *fakeRiscVCommandRepository) PublishDraft(_ context.Context, id int64, _ int64, input domain.PublishRiscVCommandDraft) (domain.RiscVCommand, error) {
	if fake.draft == nil || fake.draft.ID != id {
		return domain.RiscVCommand{}, repository.ErrRiscVCommandNotFound
	}
	command := *fake.draft
	command.AssemblyExample = input.AssemblyExample
	command.Description = input.Description
	command.Format = input.Format
	command.OperandCount = input.OperandCount
	command.LengthBits = input.LengthBits
	command.Status = domain.RiscVCommandPublished
	fake.draft = nil
	return command, nil
}

func (fake *fakeRiscVCommandRepository) SoftDelete(_ context.Context, id int64) error {
	if id <= 0 {
		return repository.ErrRiscVCommandNotFound
	}
	fake.deleted = id
	return nil
}

type fakeMediaStorage struct {
	putDirectories []string
	removedKeys    []string
}

func (fake *fakeMediaStorage) Put(_ context.Context, directory string, extension string, _ string, reader io.Reader, _ int64) (string, error) {
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return "", err
	}
	fake.putDirectories = append(fake.putDirectories, directory)
	return fmt.Sprintf("%s/generated%s", directory, extension), nil
}

func (fake *fakeMediaStorage) Remove(_ context.Context, key string) error {
	fake.removedKeys = append(fake.removedKeys, key)
	return nil
}

func TestRiscVCommandServiceFeedAndRange(t *testing.T) {
	fakeRepository := &fakeRiscVCommandRepository{commands: sampleCommands()}
	service := NewRiscVCommandService(fakeRepository, &fakeMediaStorage{})

	first, next, err := service.GetFeedRiscVCommand(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Mnemonic != "ADD" || next.Mnemonic != "SUB" {
		t.Fatalf("получена последовательность %s -> %s, ожидалась ADD -> SUB", first.Mnemonic, next.Mnemonic)
	}

	commands, minimum, maximum, err := service.GetVisibleByOperandRange(context.Background(), 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if minimum != 2 || maximum != 3 || len(commands) != 2 {
		t.Fatalf("получен диапазон %d–%d и %d команд", minimum, maximum, len(commands))
	}
}

func TestRiscVCommandServiceRejectsDeletedCommand(t *testing.T) {
	fakeRepository := &fakeRiscVCommandRepository{commands: sampleCommands()}
	service := NewRiscVCommandService(fakeRepository, &fakeMediaStorage{})
	id := int64(77)
	_, _, err := service.GetFeedRiscVCommand(context.Background(), &id)
	if !errors.Is(err, ErrRiscVCommandNotFound) {
		t.Fatalf("получена ошибка %v, ожидалась %v", err, ErrRiscVCommandNotFound)
	}
}

func TestRiscVCommandServiceCreatesAndPublishesDraft(t *testing.T) {
	fakeRepository := &fakeRiscVCommandRepository{commands: sampleCommands()}
	fakeMedia := &fakeMediaStorage{}
	service := NewRiscVCommandService(fakeRepository, fakeMedia)

	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	mp4 := []byte("\x00\x00\x00\x14ftypmp42\x00\x00\x00\x00mp42")
	draft, err := service.CreateDraft(context.Background(), 1, domain.CreateRiscVCommandDraft{
		Mnemonic: "xor",
		Image:    domain.UploadedMedia{Size: int64(len(png)), Reader: bytes.NewReader(png)},
		Video:    domain.UploadedMedia{Size: int64(len(mp4)), Reader: bytes.NewReader(mp4)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if draft.Mnemonic != "XOR" || draft.ImageKey != "images/generated.png" || draft.VideoKey != "video/generated.mp4" {
		t.Fatalf("создан некорректный черновик: %+v", draft)
	}
	if len(fakeMedia.putDirectories) != 2 {
		t.Fatalf("в MinIO отправлено файлов: %d, ожидалось 2", len(fakeMedia.putDirectories))
	}

	published, err := service.PublishDraft(context.Background(), 1, domain.PublishRiscVCommandDraft{
		AssemblyExample: "xor x3, x1, x2",
		Description:     "Выполняет побитовое исключающее ИЛИ для двух регистров.",
		Format:          domain.InstructionFormatR,
		OperandCount:    3,
		LengthBits:      32,
	})
	if err != nil {
		t.Fatal(err)
	}
	if published.Status != domain.RiscVCommandPublished || published.AssemblyExample == "" {
		t.Fatalf("получена некорректная опубликованная команда: %+v", published)
	}
}

func TestRiscVCommandServiceValidatesPublicationFormat(t *testing.T) {
	draft := domain.RiscVCommand{ID: 12, Mnemonic: "LUI", Status: domain.RiscVCommandDraft}
	service := NewRiscVCommandService(&fakeRiscVCommandRepository{draft: &draft}, &fakeMediaStorage{})
	_, err := service.PublishDraft(context.Background(), 1, domain.PublishRiscVCommandDraft{
		AssemblyExample: "lui x5, 0x12345",
		Description:     "Загружает immediate в старшие разряды регистра.",
		Format:          domain.InstructionFormatU,
		OperandCount:    3,
		LengthBits:      32,
	})
	if !errors.Is(err, ErrInvalidRiscVCommand) {
		t.Fatalf("получена ошибка %v, ожидалась ошибка формата команды", err)
	}
}

func sampleCommands() []domain.RiscVCommand {
	return []domain.RiscVCommand{
		{ID: 1, Mnemonic: "ADD", OperandCount: 3, Status: domain.RiscVCommandPublished},
		{ID: 4, Mnemonic: "SUB", OperandCount: 3, Status: domain.RiscVCommandPublished},
		{ID: 77, Mnemonic: "OR", OperandCount: 3, Status: domain.RiscVCommandDeleted},
	}
}
