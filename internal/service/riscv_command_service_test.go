package service

import (
	"context"
	"errors"
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

func (fake *fakeRiscVCommandRepository) CreateDraft(_ context.Context, creatorID int64, input domain.CreateRiscVCommandDraft) (domain.RiscVCommand, error) {
	if fake.draft != nil {
		return domain.RiscVCommand{}, repository.ErrDraftAlreadyExists
	}
	command := domain.RiscVCommand{ID: 101, Mnemonic: input.Mnemonic, Status: domain.RiscVCommandDraft, CreatorID: creatorID, LengthBits: 0}
	fake.draft = &command
	fake.commands = append(fake.commands, command)
	return command, nil
}

func (fake *fakeRiscVCommandRepository) PublishDraft(_ context.Context, id int64, _ int64, input domain.PublishRiscVCommandDraft) (domain.RiscVCommand, error) {
	if fake.draft == nil || fake.draft.ID != id {
		return domain.RiscVCommand{}, repository.ErrRiscVCommandNotFound
	}
	command := *fake.draft
	command.Description = input.Description
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

func TestRiscVCommandServiceFeedAndRange(t *testing.T) {
	fakeRepository := &fakeRiscVCommandRepository{commands: sampleCommands()}
	service := NewRiscVCommandService(fakeRepository)

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
	service := NewRiscVCommandService(fakeRepository)
	id := int64(77)
	_, _, err := service.GetFeedRiscVCommand(context.Background(), &id)
	if !errors.Is(err, ErrRiscVCommandNotFound) {
		t.Fatalf("получена ошибка %v, ожидалась %v", err, ErrRiscVCommandNotFound)
	}
}

func TestRiscVCommandServiceCreatesAndPublishesDraft(t *testing.T) {
	fakeRepository := &fakeRiscVCommandRepository{commands: sampleCommands()}
	service := NewRiscVCommandService(fakeRepository)

	draft, err := service.CreateDraft(context.Background(), 1, domain.CreateRiscVCommandDraft{
		Mnemonic: "xor",
	})
	if err != nil {
		t.Fatal(err)
	}
	if draft.Mnemonic != "XOR" || draft.ImageKey != "" || draft.VideoKey != "" {
		t.Fatalf("создан некорректный черновик: %+v", draft)
	}

	published, err := service.PublishDraft(context.Background(), 1, domain.PublishRiscVCommandDraft{
		Description:  "Выполняет побитовое исключающее ИЛИ для двух регистров.",
		OperandCount: 3,
		LengthBits:   32,
	})
	if err != nil {
		t.Fatal(err)
	}
	if published.Status != domain.RiscVCommandPublished || published.Description == "" {
		t.Fatalf("получена некорректная опубликованная команда: %+v", published)
	}
}

func TestRiscVCommandServiceValidatesOperandCount(t *testing.T) {
	draft := domain.RiscVCommand{ID: 12, Mnemonic: "LUI", Status: domain.RiscVCommandDraft}
	service := NewRiscVCommandService(&fakeRiscVCommandRepository{draft: &draft})
	_, err := service.PublishDraft(context.Background(), 1, domain.PublishRiscVCommandDraft{
		Description:  "Загружает immediate в старшие разряды регистра.",
		OperandCount: 4,
		LengthBits:   32,
	})
	if !errors.Is(err, ErrInvalidRiscVCommand) {
		t.Fatalf("получена ошибка %v, ожидалась ошибка количества операндов", err)
	}
}

func sampleCommands() []domain.RiscVCommand {
	return []domain.RiscVCommand{
		{ID: 1, Mnemonic: "ADD", OperandCount: 3, Status: domain.RiscVCommandPublished},
		{ID: 4, Mnemonic: "SUB", OperandCount: 3, Status: domain.RiscVCommandPublished},
		{ID: 77, Mnemonic: "OR", OperandCount: 3, Status: domain.RiscVCommandDeleted},
	}
}
