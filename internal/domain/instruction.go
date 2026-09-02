package domain

type InstructionStatus string

const (
	InstructionDraft     InstructionStatus = "draft"
	InstructionPublished InstructionStatus = "published"
	InstructionDeleted   InstructionStatus = "deleted"
)

// Instruction — команда RISC-V из коллекции в памяти для первой лабораторной работы.
type Instruction struct {
	ID           int
	Mnemonic     string
	Description  string
	Format       string
	OperandCount int
	ImageURL     string
	VideoURL     string
	Status       InstructionStatus
	LikedUserIDs []int
}

func (instruction Instruction) LikeCount() int {
	return len(instruction.LikedUserIDs)
}
