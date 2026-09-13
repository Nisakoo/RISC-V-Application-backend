package handler

import "riscv-lab/internal/domain"

type activePage string
type pageClass string

const (
	activePageFeed  activePage = "feed"
	activePageDraft activePage = "draft"
	activePageGrid  activePage = "grid"
	pageClassFeed   pageClass  = "feed-page"
	pageClassPlain  pageClass  = "plain-page"
)

type layoutView struct {
	Title      string
	ActivePage activePage
}

type instructionView struct {
	ID              int
	Mnemonic        string
	AssemblyExample string
	Description     string
	Format          domain.InstructionFormat
	OperandCount    int
	ImageURL        string
	VideoURL        string
	LikeCount       int
}

type feedPageView struct {
	Layout            layoutView
	Instruction       instructionView
	NextInstructionID int
}

type draftPageView struct {
	Layout      layoutView
	Instruction instructionView
}

type gridPageView struct {
	Layout              layoutView
	Instructions        []instructionView
	MinimumOperandCount int
	MaximumOperandCount int
	RangeStart          int
	RangeEnd            int
}

type errorPageView struct {
	Layout      layoutView
	Heading     string
	Message     string
	PageClass   pageClass
	ActionURL   string
	ActionLabel string
}

func newErrorPageView(heading string, message string, activePage activePage, pageClass pageClass, actionURL string, actionLabel string) errorPageView {
	return errorPageView{
		Layout:      newLayoutView(heading+" · RISC-V", activePage),
		Heading:     heading,
		Message:     message,
		PageClass:   pageClass,
		ActionURL:   actionURL,
		ActionLabel: actionLabel,
	}
}

func newLayoutView(title string, activePage activePage) layoutView {
	return layoutView{
		Title:      title,
		ActivePage: activePage,
	}
}
