package edgetts

import "fmt"

const (
	// 基础 URL
	BaseURL            = "speech.platform.bing.com/consumer/speech/synthesize/readaloud"
	TrustedClientToken = "6A5AA1D4EAFF4E9FB37E23D68491D6F4"

	// WebSocket 和语音列表 URL
	WSSURL    = "wss://" + BaseURL + "/edge/v1?TrustedClientToken=" + TrustedClientToken
	VoiceList = "https://" + BaseURL + "/voices/list?trustedclienttoken=" + TrustedClientToken

	// 默认语音
	DefaultVoice = "en-US-EmmaMultilingualNeural"

	// Chrome 版本信息
	ChromiumFullVersion  = "143.0.3650.75"
	ChromiumMajorVersion = "143"
	SecMSGECVersion      = "1-" + ChromiumFullVersion

	// Windows 纪元偏移量（秒）
	WinEpoch = 11644473600
	// 秒到纳秒
	SToNS = 1e9
)

var (
	// 基础 HTTP headers
	BaseHeaders = map[string]string{
		"User-Agent":      fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s.0.0.0 Safari/537.36 Edg/%s.0.0.0", ChromiumMajorVersion, ChromiumMajorVersion),
		"Accept-Encoding": "gzip, deflate, br, zstd",
		"Accept-Language": "en-US,en;q=0.9",
	}

	// WebSocket headers (注意: Sec-WebSocket-Version 由 gorilla/websocket 自动设置)
	WSSHeaders = map[string]string{
		"Pragma":       "no-cache",
		"Cache-Control": "no-cache",
		"Origin":       "chrome-extension://jdiccldimpdaibmpdkjnbmckianbfold",
	}

	// 语音列表请求 headers
	VoiceHeaders = map[string]string{
		"Authority":       "speech.platform.bing.com",
		"Sec-CH-UA":       fmt.Sprintf(`" Not;A Brand";v="99", "Microsoft Edge";v="%s", "Chromium";v="%s"`, ChromiumMajorVersion, ChromiumMajorVersion),
		"Sec-CH-UA-Mobile": "?0",
		"Accept":          "*/*",
		"Sec-Fetch-Site":  "none",
		"Sec-Fetch-Mode":  "cors",
		"Sec-Fetch-Dest":  "empty",
	}
)

func init() {
	// 合并 BaseHeaders 到 WSSHeaders
	for k, v := range BaseHeaders {
		WSSHeaders[k] = v
	}
	// 合并 BaseHeaders 到 VoiceHeaders
	for k, v := range BaseHeaders {
		VoiceHeaders[k] = v
	}
}
