package edgetts

import "errors"

var (
	// ErrUnknownResponse 收到未知响应
	ErrUnknownResponse = errors.New("unknown response from server")

	// ErrUnexpectedResponse 收到意外响应
	ErrUnexpectedResponse = errors.New("unexpected response from server")

	// ErrNoAudioReceived 没有收到音频
	ErrNoAudioReceived = errors.New("no audio received from server")

	// ErrWebSocket WebSocket 错误
	ErrWebSocket = errors.New("websocket error")

	// ErrSkewAdjustment 时钟偏移调整错误
	ErrSkewAdjustment = errors.New("clock skew adjustment error")

	// ErrInvalidVoice 无效的语音
	ErrInvalidVoice = errors.New("invalid voice format")

	// ErrInvalidRate 无效的语速
	ErrInvalidRate = errors.New("invalid rate format")

	// ErrInvalidVolume 无效的音量
	ErrInvalidVolume = errors.New("invalid volume format")

	// ErrInvalidPitch 无效的音调
	ErrInvalidPitch = errors.New("invalid pitch format")

	// ErrStreamAlreadyCalled stream 已经被调用
	ErrStreamAlreadyCalled = errors.New("stream can only be called once")
)
