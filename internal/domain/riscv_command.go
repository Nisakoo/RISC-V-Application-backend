package domain

import (
	"io"
	"time"
)

type InstructionFormat string

const (
	InstructionFormatR InstructionFormat = "R"
	InstructionFormatI InstructionFormat = "I"
	InstructionFormatS InstructionFormat = "S"
	InstructionFormatB InstructionFormat = "B"
	InstructionFormatU InstructionFormat = "U"
	InstructionFormatJ InstructionFormat = "J"
)

type RiscVCommandStatus string

const (
	RiscVCommandDraft     RiscVCommandStatus = "draft"
	RiscVCommandPublished RiscVCommandStatus = "published"
	RiscVCommandDeleted   RiscVCommandStatus = "deleted"
)

// RiscVCommand описывает команду каталога RISC-V и её состояние публикации.
type RiscVCommand struct {
	ID              int64
	Mnemonic        string
	AssemblyExample string
	Description     string
	Format          InstructionFormat
	OperandCount    int
	LengthBits      int
	ImageKey        string
	VideoKey        string
	Status          RiscVCommandStatus
	CreatedAt       time.Time
	PublishedAt     *time.Time
	CreatorID       int64
	LikeCount       int64
}

// UploadedMedia содержит проверяемые метаданные и поток загруженного файла.
type UploadedMedia struct {
	OriginalName string
	ContentType  string
	Size         int64
	Reader       io.Reader
}

type CreateRiscVCommandDraft struct {
	Mnemonic string
	Image    UploadedMedia
	Video    UploadedMedia
}

type PublishRiscVCommandDraft struct {
	AssemblyExample string
	Description     string
	Format          InstructionFormat
	OperandCount    int
	LengthBits      int
}
