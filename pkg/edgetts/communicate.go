package edgetts

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

// CommunicateOption 通信选项
type CommunicateOption func(*Communicate)

// WithRate 设置语速
func WithRate(rate string) CommunicateOption {
	return func(c *Communicate) {
		c.ttsConfig.Rate = rate
	}
}

// WithVolume 设置音量
func WithVolume(volume string) CommunicateOption {
	return func(c *Communicate) {
		c.ttsConfig.Volume = volume
	}
}

// WithPitch 设置音调
func WithPitch(pitch string) CommunicateOption {
	return func(c *Communicate) {
		c.ttsConfig.Pitch = pitch
	}
}

// WithProxy 设置代理
func WithProxy(proxy string) CommunicateOption {
	return func(c *Communicate) {
		c.proxy = proxy
	}
}

// WithConnectTimeout 设置连接超时
func WithConnectTimeout(timeout time.Duration) CommunicateOption {
	return func(c *Communicate) {
		c.connectTimeout = timeout
	}
}

// WithReceiveTimeout 设置接收超时
func WithReceiveTimeout(timeout time.Duration) CommunicateOption {
	return func(c *Communicate) {
		c.receiveTimeout = timeout
	}
}

// WithBoundary 设置边界类型
func WithBoundary(boundary string) CommunicateOption {
	return func(c *Communicate) {
		c.ttsConfig.Boundary = boundary
	}
}

// Communicate 与 TTS 服务通信
type Communicate struct {
	ttsConfig      *TTSConfig
	texts          [][]byte
	proxy          string
	connectTimeout time.Duration
	receiveTimeout time.Duration
	state          *CommunicateState
}

// NewCommunicate 创建新的通信实例
func NewCommunicate(text string, voice string, opts ...CommunicateOption) (*Communicate, error) {
	if voice == "" {
		voice = DefaultVoice
	}

	c := &Communicate{
		ttsConfig: &TTSConfig{
			Voice:    voice,
			Rate:     "+0%",
			Volume:   "+0%",
			Pitch:    "+0Hz",
			Boundary: "SentenceBoundary",
		},
		connectTimeout: 10 * time.Second,
		receiveTimeout: 60 * time.Second,
		state: &CommunicateState{
			PartialText:        nil,
			OffsetCompensation: 0,
			LastDurationOffset: 0,
			StreamWasCalled:    false,
		},
	}

	// 应用选项
	for _, opt := range opts {
		opt(c)
	}

	// 验证配置
	if err := ValidateTTSConfig(c.ttsConfig); err != nil {
		return nil, err
	}

	// 处理文本：移除不兼容字符，转义，按字节分割
	cleanText := RemoveIncompatibleCharacters(text)
	escapedText := EscapeXML(cleanText)
	c.texts = SplitTextByByteLength(escapedText, 4096)

	return c, nil
}

// parseMetadata 解析元数据
func (c *Communicate) parseMetadata(data []byte) (*TTSChunk, error) {
	var resp MetadataResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	for _, meta := range resp.Metadata {
		if meta.Type == "WordBoundary" || meta.Type == "SentenceBoundary" {
			currentOffset := meta.Data.Offset + c.state.OffsetCompensation
			return &TTSChunk{
				Type:     meta.Type,
				Offset:   currentOffset,
				Duration: meta.Data.Duration,
				Text:     UnescapeXML(meta.Data.Text.Text),
			}, nil
		}
		if meta.Type == "SessionEnd" {
			continue
		}
		return nil, fmt.Errorf("%w: unknown metadata type: %s", ErrUnknownResponse, meta.Type)
	}

	return nil, fmt.Errorf("%w: no WordBoundary metadata found", ErrUnexpectedResponse)
}

// stream 内部流处理
func (c *Communicate) stream(ctx context.Context) (<-chan TTSChunk, <-chan error) {
	chunkCh := make(chan TTSChunk, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		drm := GetDRM()

		// 构建 WebSocket URL
		wsURL := fmt.Sprintf("%s&ConnectionId=%s&Sec-MS-GEC=%s&Sec-MS-GEC-Version=%s",
			WSSURL, ConnectID(), drm.GenerateSecMSGEC(), SecMSGECVersion)

		// 设置 WebSocket headers
		headers := http.Header{}
		for k, v := range HeadersWithMUID(WSSHeaders) {
			headers.Set(k, v)
		}

		// 连接 WebSocket
		dialer := websocket.Dialer{
			HandshakeTimeout: c.connectTimeout,
		}

		conn, _, err := dialer.DialContext(ctx, wsURL, headers)
		if err != nil {
			errCh <- fmt.Errorf("websocket dial error: %w", err)
			return
		}
		defer conn.Close()

		// 设置读取超时
		conn.SetReadDeadline(time.Now().Add(c.receiveTimeout))

		// 发送配置请求
		wordBoundary := c.ttsConfig.Boundary == "WordBoundary"
		wd := "false"
		sq := "true"
		if wordBoundary {
			wd = "true"
			sq = "false"
		}

		configMsg := fmt.Sprintf("X-Timestamp:%s\r\n"+
			"Content-Type:application/json; charset=utf-8\r\n"+
			"Path:speech.config\r\n\r\n"+
			`{"context":{"synthesis":{"audio":{"metadataoptions":`+
			`{"sentenceBoundaryEnabled":"%s","wordBoundaryEnabled":"%s"},`+
			`"outputFormat":"audio-24khz-48kbitrate-mono-mp3"}}}}`+"\r\n",
			DateToString(), sq, wd)

		if err := conn.WriteMessage(websocket.TextMessage, []byte(configMsg)); err != nil {
			errCh <- fmt.Errorf("write config error: %w", err)
			return
		}

		// 发送 SSML 请求
		ssml := MKSSML(c.ttsConfig, string(c.state.PartialText))
		ssmlMsg := SSMLHeadersPlusData(ConnectID(), DateToString(), ssml)

		if err := conn.WriteMessage(websocket.TextMessage, []byte(ssmlMsg)); err != nil {
			errCh <- fmt.Errorf("write ssml error: %w", err)
			return
		}

		audioReceived := false

		// 读取响应
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			conn.SetReadDeadline(time.Now().Add(c.receiveTimeout))
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					break
				}
				errCh <- fmt.Errorf("read message error: %w", err)
				return
			}

			if msgType == websocket.TextMessage {
				// 找到 header 和 data 的分隔点
				headerEnd := -1
				for i := 0; i < len(data)-3; i++ {
					if data[i] == '\r' && data[i+1] == '\n' && data[i+2] == '\r' && data[i+3] == '\n' {
						headerEnd = i
						break
					}
				}

				if headerEnd < 0 {
					continue
				}

				headers, body := GetHeadersAndData(data, headerEnd)
				path := headers["Path"]

				switch path {
				case "audio.metadata":
					parsed, err := c.parseMetadata(body)
					if err != nil {
						errCh <- err
						return
					}
					chunkCh <- *parsed
					c.state.LastDurationOffset = parsed.Offset + parsed.Duration

				case "turn.end":
					c.state.OffsetCompensation = c.state.LastDurationOffset
					c.state.OffsetCompensation += 8_750_000
					goto done

				case "response", "turn.start":
					// 忽略

				default:
					errCh <- fmt.Errorf("%w: unknown path: %s", ErrUnknownResponse, path)
					return
				}

			} else if msgType == websocket.BinaryMessage {
				if len(data) < 2 {
					errCh <- fmt.Errorf("%w: binary message missing header length", ErrUnexpectedResponse)
					return
				}

				headerLength := int(binary.BigEndian.Uint16(data[:2]))
				if headerLength+2 > len(data) {
					errCh <- fmt.Errorf("%w: header length > data length", ErrUnexpectedResponse)
					return
				}

				// 跳过前 2 字节（长度），解析 headers 和 body
				headers, body := GetHeadersAndData(data[2:], headerLength)

				if headers["Path"] != "audio" {
					errCh <- fmt.Errorf("%w: binary message path is not audio", ErrUnexpectedResponse)
					return
				}

				contentType := headers["Content-Type"]
				if contentType != "audio/mpeg" && contentType != "" {
					errCh <- fmt.Errorf("%w: unexpected content type: %s", ErrUnexpectedResponse, contentType)
					return
				}

				if contentType == "" {
					if len(body) == 0 {
						continue
					}
					errCh <- fmt.Errorf("%w: no content type but has data", ErrUnexpectedResponse)
					return
				}

				if len(body) == 0 {
					errCh <- fmt.Errorf("%w: audio content type but no data", ErrUnexpectedResponse)
					return
				}

				audioReceived = true
				chunkCh <- TTSChunk{
					Type: "audio",
					Data: body,
				}
			}
		}

	done:
		if !audioReceived {
			errCh <- ErrNoAudioReceived
		}
	}()

	return chunkCh, errCh
}

// Stream 流式获取音频和元数据
func (c *Communicate) Stream(ctx context.Context) (<-chan TTSChunk, <-chan error) {
	chunkCh := make(chan TTSChunk, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		if c.state.StreamWasCalled {
			errCh <- ErrStreamAlreadyCalled
			return
		}
		c.state.StreamWasCalled = true

		for _, text := range c.texts {
			c.state.PartialText = text

			innerChunkCh, innerErrCh := c.stream(ctx)

			// 转发所有 chunks
		loop:
			for {
				select {
				case chunk, ok := <-innerChunkCh:
					if !ok {
						break loop
					}
					chunkCh <- chunk
				case err, ok := <-innerErrCh:
					if ok && err != nil {
						errCh <- err
						return
					}
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				}
			}

			// 检查错误
			select {
			case err := <-innerErrCh:
				if err != nil {
					errCh <- err
					return
				}
			default:
			}
		}
	}()

	return chunkCh, errCh
}

// Save 保存音频和元数据到文件
func (c *Communicate) Save(ctx context.Context, audioFname string, metadataFname string) error {
	audioFile, err := os.Create(audioFname)
	if err != nil {
		return err
	}
	defer audioFile.Close()

	var metadataFile *os.File
	if metadataFname != "" {
		metadataFile, err = os.Create(metadataFname)
		if err != nil {
			return err
		}
		defer metadataFile.Close()
	}

	chunkCh, errCh := c.Stream(ctx)

	for {
		select {
		case chunk, ok := <-chunkCh:
			if !ok {
				return nil
			}
			if chunk.Type == "audio" {
				if _, err := audioFile.Write(chunk.Data); err != nil {
					return err
				}
			} else if metadataFile != nil && (chunk.Type == "WordBoundary" || chunk.Type == "SentenceBoundary") {
				data, _ := json.Marshal(chunk)
				metadataFile.Write(data)
				metadataFile.WriteString("\n")
			}
		case err := <-errCh:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// StreamSync 同步流式接口（使用 channel）
func (c *Communicate) StreamSync(ctx context.Context) ([]TTSChunk, error) {
	var chunks []TTSChunk

	chunkCh, errCh := c.Stream(ctx)

	for {
		select {
		case chunk, ok := <-chunkCh:
			if !ok {
				return chunks, nil
			}
			chunks = append(chunks, chunk)
		case err := <-errCh:
			if err != nil {
				return chunks, err
			}
		case <-ctx.Done():
			return chunks, ctx.Err()
		}
	}
}

// SaveSync 同步保存接口
func (c *Communicate) SaveSync(audioFname string, metadataFname string) error {
	return c.Save(context.Background(), audioFname, metadataFname)
}

// StreamToWriter 流式写入到 writer
func (c *Communicate) StreamToWriter(ctx context.Context, w io.Writer, submaker *SubMaker) error {
	chunkCh, errCh := c.Stream(ctx)

	for {
		select {
		case chunk, ok := <-chunkCh:
			if !ok {
				return nil
			}
			if chunk.Type == "audio" {
				if _, err := w.Write(chunk.Data); err != nil {
					return err
				}
			} else if submaker != nil && (chunk.Type == "WordBoundary" || chunk.Type == "SentenceBoundary") {
				if err := submaker.Feed(chunk); err != nil {
					return err
				}
			}
		case err := <-errCh:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
