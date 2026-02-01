package edgetts

// TTSChunk 表示 TTS 返回的数据块
type TTSChunk struct {
	Type     string  // "audio", "WordBoundary", "SentenceBoundary"
	Data     []byte  // 仅用于 audio 类型
	Duration float64 // 仅用于 WordBoundary 和 SentenceBoundary
	Offset   float64 // 仅用于 WordBoundary 和 SentenceBoundary
	Text     string  // 仅用于 WordBoundary 和 SentenceBoundary
}

// VoiceTag 语音标签
type VoiceTag struct {
	ContentCategories   []string `json:"ContentCategories"`
	VoicePersonalities []string `json:"VoicePersonalities"`
}

// Voice 语音信息
type Voice struct {
	Name           string   `json:"Name"`
	ShortName      string   `json:"ShortName"`
	Gender         string   `json:"Gender"`
	Locale         string   `json:"Locale"`
	SuggestedCodec string   `json:"SuggestedCodec"`
	FriendlyName   string   `json:"FriendlyName"`
	Status         string   `json:"Status"`
	VoiceTag       VoiceTag `json:"VoiceTag"`
	Language       string   `json:"Language,omitempty"` // VoicesManager 添加的字段
}

// TTSConfig TTS 配置
type TTSConfig struct {
	Voice    string
	Rate     string
	Volume   string
	Pitch    string
	Boundary string // "WordBoundary" 或 "SentenceBoundary"
}

// CommunicateState 通信状态
type CommunicateState struct {
	PartialText        []byte
	OffsetCompensation float64
	LastDurationOffset float64
	StreamWasCalled    bool
}

// Metadata 响应元数据
type Metadata struct {
	Type string `json:"Type"`
	Data struct {
		Offset   float64 `json:"Offset"`
		Duration float64 `json:"Duration"`
		Text     struct {
			Text string `json:"Text"`
		} `json:"text"`
	} `json:"Data"`
}

// MetadataResponse 元数据响应
type MetadataResponse struct {
	Metadata []Metadata `json:"Metadata"`
}
