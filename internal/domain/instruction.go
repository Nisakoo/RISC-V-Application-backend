package domain

type InstructionStatus string

const (
	InstructionDraft     InstructionStatus = "draft"
	InstructionPublished InstructionStatus = "published"
	InstructionDeleted   InstructionStatus = "deleted"
)

// Instruction is a RISC-V command from the in-memory collection used in lab 1.
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
