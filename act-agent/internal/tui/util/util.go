package util

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

func CmdHandler(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}

func ReportError(err error) tea.Cmd {
	return CmdHandler(InfoMsg{
		Type: InfoTypeError,
		Msg:  err.Error(),
	})
}

type InfoType int

const (
	InfoTypeInfo InfoType = iota
	InfoTypeWarn
	InfoTypeError
)

func ReportInfo(info string) tea.Cmd {
	return CmdHandler(InfoMsg{
		Type: InfoTypeInfo,
		Msg:  info,
	})
}

func ReportWarn(warn string) tea.Cmd {
	return CmdHandler(InfoMsg{
		Type: InfoTypeWarn,
		Msg:  warn,
	})
}

type (
	InfoMsg struct {
		Type InfoType
		Msg  string
		TTL  time.Duration
	}
	// ClearStatusMsg retires the current status banner. Gen carries the
	// generation of the InfoMsg whose timer scheduled it; receivers ignore
	// a stale tick (Gen older than the current banner's generation) so an
	// earlier TTL can't cut a newer message short.
	ClearStatusMsg struct{ Gen uint64 }
)

func Clamp(v, low, high int) int {
	if high < low {
		low, high = high, low
	}
	return min(high, max(low, v))
}
