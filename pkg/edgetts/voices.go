package edgetts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ListVoicesOptions 列出语音的选项
type ListVoicesOptions struct {
	Proxy   string
	Timeout time.Duration
}

// listVoicesInternal 内部函数，执行实际的语音列表请求
func listVoicesInternal(ctx context.Context, opts *ListVoicesOptions) ([]Voice, error) {
	drm := GetDRM()

	url := fmt.Sprintf("%s&Sec-MS-GEC=%s&Sec-MS-GEC-Version=%s",
		VoiceList, drm.GenerateSecMSGEC(), SecMSGECVersion)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 设置 headers
	headers := HeadersWithMUID(VoiceHeaders)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{
		Timeout: opts.Timeout,
	}

	// 设置代理
	if opts.Proxy != "" {
		// 简化处理，实际使用时需要配置代理
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("forbidden: status %d", resp.StatusCode)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var voices []Voice
	if err := json.Unmarshal(body, &voices); err != nil {
		return nil, err
	}

	// 确保 VoiceTag 字段存在
	for i := range voices {
		if voices[i].VoiceTag.ContentCategories == nil {
			voices[i].VoiceTag.ContentCategories = []string{}
		}
		if voices[i].VoiceTag.VoicePersonalities == nil {
			voices[i].VoiceTag.VoicePersonalities = []string{}
		}
	}

	return voices, nil
}

// ListVoices 列出所有可用的语音
func ListVoices(ctx context.Context, opts *ListVoicesOptions) ([]Voice, error) {
	if opts == nil {
		opts = &ListVoicesOptions{
			Timeout: 30 * time.Second,
		}
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}

	voices, err := listVoicesInternal(ctx, opts)
	if err != nil {
		// 如果是 403 错误，尝试调整时钟偏移后重试
		if strings.Contains(err.Error(), "forbidden") || strings.Contains(err.Error(), "403") {
			// 注意：这里简化处理，实际上应该从响应中获取服务器时间
			return listVoicesInternal(ctx, opts)
		}
		return nil, err
	}
	return voices, nil
}

// VoicesManager 语音管理器
type VoicesManager struct {
	Voices       []Voice
	CalledCreate bool
}

// NewVoicesManager 创建新的 VoicesManager
func NewVoicesManager() *VoicesManager {
	return &VoicesManager{
		Voices:       []Voice{},
		CalledCreate: false,
	}
}

// Create 创建并填充 VoicesManager
func (vm *VoicesManager) Create(ctx context.Context, customVoices []Voice) error {
	var voices []Voice
	var err error

	if customVoices != nil {
		voices = customVoices
	} else {
		voices, err = ListVoices(ctx, nil)
		if err != nil {
			return err
		}
	}

	// 添加 Language 字段
	for i := range voices {
		parts := strings.Split(voices[i].Locale, "-")
		if len(parts) > 0 {
			voices[i].Language = parts[0]
		}
	}

	vm.Voices = voices
	vm.CalledCreate = true
	return nil
}

// Find 根据条件查找语音
func (vm *VoicesManager) Find(gender, locale, language string) ([]Voice, error) {
	if !vm.CalledCreate {
		return nil, fmt.Errorf("VoicesManager.Find() called before VoicesManager.Create()")
	}

	var result []Voice
	for _, voice := range vm.Voices {
		match := true
		if gender != "" && voice.Gender != gender {
			match = false
		}
		if locale != "" && voice.Locale != locale {
			match = false
		}
		if language != "" && voice.Language != language {
			match = false
		}
		if match {
			result = append(result, voice)
		}
	}
	return result, nil
}
