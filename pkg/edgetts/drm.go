package edgetts

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DRM 处理 DRM 操作和时钟偏移校正
type DRM struct {
	mu               sync.RWMutex
	clockSkewSeconds float64
}

var globalDRM = &DRM{}

// AdjClockSkewSeconds 调整时钟偏移
func (d *DRM) AdjClockSkewSeconds(skewSeconds float64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.clockSkewSeconds += skewSeconds
}

// GetUnixTimestamp 获取带时钟偏移校正的 Unix 时间戳
func (d *DRM) GetUnixTimestamp() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return float64(time.Now().UTC().Unix()) + d.clockSkewSeconds
}

// ParseRFC2616Date 解析 RFC 2616 格式的日期
func ParseRFC2616Date(dateStr string) (float64, error) {
	t, err := time.Parse(time.RFC1123, dateStr)
	if err != nil {
		// 尝试其他格式
		t, err = time.Parse("Mon, 02 Jan 2006 15:04:05 GMT", dateStr)
		if err != nil {
			return 0, err
		}
	}
	return float64(t.Unix()), nil
}

// HandleClientResponseError 处理客户端响应错误，调整时钟偏移
func (d *DRM) HandleClientResponseError(resp *http.Response) error {
	if resp == nil || resp.Header == nil {
		return fmt.Errorf("%w: no server date in headers", ErrSkewAdjustment)
	}

	serverDate := resp.Header.Get("Date")
	if serverDate == "" {
		return fmt.Errorf("%w: no server date in headers", ErrSkewAdjustment)
	}

	serverTimestamp, err := ParseRFC2616Date(serverDate)
	if err != nil {
		return fmt.Errorf("%w: failed to parse server date: %s", ErrSkewAdjustment, serverDate)
	}

	clientTimestamp := d.GetUnixTimestamp()
	d.AdjClockSkewSeconds(serverTimestamp - clientTimestamp)
	return nil
}

// GenerateSecMSGEC 生成 Sec-MS-GEC token
func (d *DRM) GenerateSecMSGEC() string {
	// 获取带时钟偏移校正的时间戳
	ticks := d.GetUnixTimestamp()

	// 转换为 Windows 文件时间纪元
	ticks += WinEpoch

	// 向下取整到最近的 5 分钟（300 秒）
	ticks -= float64(int64(ticks) % 300)

	// 转换为 100 纳秒间隔（Windows 文件时间格式）
	ticks *= SToNS / 100

	// 创建要哈希的字符串
	strToHash := fmt.Sprintf("%.0f%s", ticks, TrustedClientToken)

	// 计算 SHA256 哈希并返回大写的十六进制
	hash := sha256.Sum256([]byte(strToHash))
	return strings.ToUpper(hex.EncodeToString(hash[:]))
}

// GenerateMUID 生成随机 MUID
func GenerateMUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

// HeadersWithMUID 返回带有 MUID cookie 的 headers
func HeadersWithMUID(headers map[string]string) map[string]string {
	combined := make(map[string]string)
	for k, v := range headers {
		combined[k] = v
	}
	combined["Cookie"] = fmt.Sprintf("muid=%s;", GenerateMUID())
	return combined
}

// GetDRM 获取全局 DRM 实例
func GetDRM() *DRM {
	return globalDRM
}
