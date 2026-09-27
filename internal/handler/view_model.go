package handler

import (
	"riscv-lab/internal/route"
)

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
	FeedURL    string
	AddURL     string
	PublishURL string
	DeleteURL  string
	GridURL    string
}

type riscvCommandView struct {
	ID           int64
	Mnemonic     string
	Description  string
	OperandCount int
	LengthBits   int
	ImageURL     string
	VideoURL     string
	LikeCount    int64
}

type feedPageView struct {
	Layout              layoutView
	RiscVCommand        riscvCommandView
	NextRiscVCommandURL string
}

type draftPageView struct {
	Layout       layoutView
	HasDraft     bool
	RiscVCommand riscvCommandView
	ErrorMessage string
}

type gridPageView struct {
	Layout              layoutView
	RiscVCommands       []riscvCommandView
	MinimumOperandCount int
	MaximumOperandCount int
	RangeStart          int
	RangeEnd            int
	ErrorMessage        string
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
		FeedURL:    route.Feed,
		AddURL:     route.Add,
		PublishURL: route.Publish,
		DeleteURL:  route.Delete,
		GridURL:    route.Grid,
	}
}
