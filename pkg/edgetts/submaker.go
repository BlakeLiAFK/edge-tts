package edgetts

import (
	"fmt"
	"time"
)

// SubMaker 字幕生成器
type SubMaker struct {
	Cues     []Subtitle
	CueType  string // "WordBoundary" 或 "SentenceBoundary"
}

// NewSubMaker 创建新的字幕生成器
func NewSubMaker() *SubMaker {
	return &SubMaker{
		Cues:    []Subtitle{},
		CueType: "",
	}
}

// Feed 将 WordBoundary 或 SentenceBoundary 消息喂入字幕生成器
func (sm *SubMaker) Feed(msg TTSChunk) error {
	if msg.Type != "WordBoundary" && msg.Type != "SentenceBoundary" {
		return fmt.Errorf("invalid message type, expected 'WordBoundary' or 'SentenceBoundary', got '%s'", msg.Type)
	}

	if sm.CueType == "" {
		sm.CueType = msg.Type
	} else if sm.CueType != msg.Type {
		return fmt.Errorf("expected message type '%s', but got '%s'", sm.CueType, msg.Type)
	}

	// Offset 和 Duration 以 100 纳秒为单位，转换为微秒
	startMicros := msg.Offset / 10
	endMicros := (msg.Offset + msg.Duration) / 10

	subtitle := Subtitle{
		Index:   len(sm.Cues) + 1,
		Start:   time.Duration(startMicros) * time.Microsecond,
		End:     time.Duration(endMicros) * time.Microsecond,
		Content: msg.Text,
	}

	sm.Cues = append(sm.Cues, subtitle)
	return nil
}

// GetSRT 获取 SRT 格式的字幕
func (sm *SubMaker) GetSRT() string {
	return ComposeSRT(sm.Cues, true, 1, "")
}

// String 返回 SRT 格式的字幕
func (sm *SubMaker) String() string {
	return sm.GetSRT()
}
