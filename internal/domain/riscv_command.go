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
	ID          int64
	Mnemonic    string
	Description string
	// DescriptionEn — краткое описание на английском для поиска по изображению
	DescriptionEn string
	OperandCount  int
	LengthBits    int
	ImageKey      string
	VideoKey      string
	Status        RiscVCommandStatus
	CreatedAt     time.Time
	PublishedAt   *time.Time
	CreatorID     int64
	CreatorLogin  string
	LikeCount     int64
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

// FeedBatch описывает порцию ленты сессии: случайный набор команд, сохранённый в Redis.
// Порядковые номера продолжаются от порции к порции в пределах одной сессии.
type FeedBatch struct {
	SessionID     string  `json:"session_id"`
	BatchNumber   int     `json:"batch_number"`
	FirstPosition int     `json:"first_position"`
	IDs           []int64 `json:"ids"`
}

// FeedBatchItem — команда порции ленты с порядковым номером в сессии и признаком лайка пользователя
type FeedBatchItem struct {
	Command  RiscVCommand
	Position int
	Liked    bool
}
