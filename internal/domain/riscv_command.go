package domain

import "time"

type RiscVCommandStatus string

const (
	RiscVCommandDraft     RiscVCommandStatus = "draft"
	RiscVCommandPublished RiscVCommandStatus = "published"
	RiscVCommandDeleted   RiscVCommandStatus = "deleted"
)

// RiscVCommand описывает команду каталога RISC-V и её состояние публикации.
type RiscVCommand struct {
	ID           int64
	Mnemonic     string
	Description  string
	OperandCount int
	LengthBits   int
	ImageKey     string
	VideoKey     string
	Status       RiscVCommandStatus
	CreatedAt    time.Time
	PublishedAt  *time.Time
	CreatorID    int64
	CreatorLogin string
	LikeCount    int64
}

type CreateRiscVCommandDraft struct {
	Mnemonic string
	ImageKey string
	VideoKey string
}

type PublishRiscVCommandDraft struct {
	Description  string `json:"description"`
	OperandCount int    `json:"operand_count"`
	LengthBits   int    `json:"length_bits"`
}
