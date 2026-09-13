package domain

type InstructionFormat string

const (
	InstructionFormatR InstructionFormat = "R"
	InstructionFormatI InstructionFormat = "I"
	InstructionFormatB InstructionFormat = "B"
	InstructionFormatU InstructionFormat = "U"
	InstructionFormatJ InstructionFormat = "J"
)

type InstructionStatus string

const (
	InstructionDraft     InstructionStatus = "draft"
	InstructionPublished InstructionStatus = "published"
	InstructionDeleted   InstructionStatus = "deleted"
)

// Instruction — команда RISC-V из коллекции в памяти для первой лабораторной работы.
type Instruction struct {
	ID              int
	Mnemonic        string
	AssemblyExample string
	Description     string
	Format          InstructionFormat
	OperandCount    int
	LengthBits      int
	ImageURL        string
	VideoURL        string
	Status          InstructionStatus
	LikedUserIDs    []int
}
