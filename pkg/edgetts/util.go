package edgetts

import (
	"bytes"
	"html"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// ConnectID 生成不带连字符的 UUID
func ConnectID() string {
	return GenerateMUID()
}

// RemoveIncompatibleCharacters 移除服务不支持的字符
func RemoveIncompatibleCharacters(s string) string {
	var result strings.Builder
	for _, r := range s {
		code := int(r)
		if (code >= 0 && code <= 8) || (code >= 11 && code <= 12) || (code >= 14 && code <= 31) {
			result.WriteRune(' ')
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// DateToString 返回 JavaScript 风格的日期字符串
func DateToString() string {
	return time.Now().UTC().Format("Mon Jan 02 2006 15:04:05 GMT+0000 (Coordinated Universal Time)")
}

// EscapeXML 转义 XML 特殊字符
func EscapeXML(s string) string {
	return html.EscapeString(s)
}

// UnescapeXML 反转义 XML 特殊字符
func UnescapeXML(s string) string {
	return html.UnescapeString(s)
}

// findLastNewlineOrSpaceWithinLimit 在限制范围内查找最后的换行符或空格
func findLastNewlineOrSpaceWithinLimit(text []byte, limit int) int {
	// 优先查找换行符
	splitAt := bytes.LastIndex(text[:limit], []byte("\n"))
	if splitAt < 0 {
		// 如果没有换行符，查找空格
		splitAt = bytes.LastIndex(text[:limit], []byte(" "))
	}
	return splitAt
}

// findSafeUTF8SplitPoint 查找安全的 UTF-8 分割点
func findSafeUTF8SplitPoint(textSegment []byte) int {
	splitAt := len(textSegment)
	for splitAt > 0 {
		if utf8.Valid(textSegment[:splitAt]) {
			return splitAt
		}
		splitAt--
	}
	return splitAt
}

// adjustSplitPointForXMLEntity 调整分割点以避免切割 XML 实体
func adjustSplitPointForXMLEntity(text []byte, splitAt int) int {
	for splitAt > 0 && bytes.Contains(text[:splitAt], []byte("&")) {
		ampIndex := bytes.LastIndex(text[:splitAt], []byte("&"))
		// 检查 & 和分割点之间是否有 ;
		if bytes.Index(text[ampIndex:splitAt], []byte(";")) != -1 {
			break
		}
		splitAt = ampIndex
	}
	return splitAt
}

// SplitTextByByteLength 按字节长度分割文本
func SplitTextByByteLength(text string, byteLength int) [][]byte {
	if byteLength <= 0 {
		return nil
	}

	textBytes := []byte(text)
	var result [][]byte

	for len(textBytes) > byteLength {
		splitAt := findLastNewlineOrSpaceWithinLimit(textBytes, byteLength)
		if splitAt < 0 {
			splitAt = findSafeUTF8SplitPoint(textBytes[:byteLength])
		}
		splitAt = adjustSplitPointForXMLEntity(textBytes, splitAt)

		if splitAt <= 0 {
			splitAt = 1
		}

		chunk := bytes.TrimSpace(textBytes[:splitAt])
		if len(chunk) > 0 {
			result = append(result, chunk)
		}

		textBytes = textBytes[splitAt:]
	}

	remaining := bytes.TrimSpace(textBytes)
	if len(remaining) > 0 {
		result = append(result, remaining)
	}

	return result
}

// MKSSML 创建 SSML 字符串
func MKSSML(tc *TTSConfig, escapedText string) string {
	return "<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='en-US'>" +
		"<voice name='" + tc.Voice + "'>" +
		"<prosody pitch='" + tc.Pitch + "' rate='" + tc.Rate + "' volume='" + tc.Volume + "'>" +
		escapedText +
		"</prosody>" +
		"</voice>" +
		"</speak>"
}

// SSMLHeadersPlusData 返回请求的 headers 和 data
func SSMLHeadersPlusData(requestID, timestamp, ssml string) string {
	return "X-RequestId:" + requestID + "\r\n" +
		"Content-Type:application/ssml+xml\r\n" +
		"X-Timestamp:" + timestamp + "Z\r\n" +
		"Path:ssml\r\n\r\n" +
		ssml
}

// ValidateTTSConfig 验证 TTS 配置
func ValidateTTSConfig(tc *TTSConfig) error {
	// 验证并转换 voice 格式
	voicePattern := regexp.MustCompile(`^([a-z]{2,})-([A-Z]{2,})-(.+Neural)$`)
	matches := voicePattern.FindStringSubmatch(tc.Voice)
	if matches != nil {
		lang := matches[1]
		region := matches[2]
		name := matches[3]

		if idx := strings.Index(name, "-"); idx != -1 {
			region = region + "-" + name[:idx]
			name = name[idx+1:]
		}

		tc.Voice = "Microsoft Server Speech Text to Speech Voice (" + lang + "-" + region + ", " + name + ")"
	}

	// 验证 voice 格式
	voiceFullPattern := regexp.MustCompile(`^Microsoft Server Speech Text to Speech Voice \(.+,.+\)$`)
	if !voiceFullPattern.MatchString(tc.Voice) {
		return ErrInvalidVoice
	}

	// 验证 rate
	ratePattern := regexp.MustCompile(`^[+-]\d+%$`)
	if !ratePattern.MatchString(tc.Rate) {
		return ErrInvalidRate
	}

	// 验证 volume
	volumePattern := regexp.MustCompile(`^[+-]\d+%$`)
	if !volumePattern.MatchString(tc.Volume) {
		return ErrInvalidVolume
	}

	// 验证 pitch
	pitchPattern := regexp.MustCompile(`^[+-]\d+Hz$`)
	if !pitchPattern.MatchString(tc.Pitch) {
		return ErrInvalidPitch
	}

	return nil
}

// GetHeadersAndData 从数据中解析 headers 和 data
func GetHeadersAndData(data []byte, headerLength int) (map[string]string, []byte) {
	headers := make(map[string]string)

	if headerLength > len(data) {
		headerLength = len(data)
	}

	headerBytes := data[:headerLength]

	lines := bytes.Split(headerBytes, []byte("\r\n"))
	for _, line := range lines {
		parts := bytes.SplitN(line, []byte(":"), 2)
		if len(parts) == 2 {
			headers[string(parts[0])] = string(parts[1])
		}
	}

	// 计算 body 的起始位置（data 已经不包含长度字段）
	bodyStart := headerLength
	if bodyStart > len(data) {
		bodyStart = len(data)
	}

	return headers, data[bodyStart:]
}
