package main

import (
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/BlakeLiAFK/edge-tts/pkg/edgetts"
)

//go:embed all:static
var staticFiles embed.FS

var voicesCache []edgetts.Voice
var voicesCacheTime time.Time

func main() {
	addr := flag.String("addr", ":8080", "监听地址")
	openBrowser := flag.Bool("open", false, "启动后自动打开浏览器")
	flag.Parse()

	// 预加载语音列表
	go preloadVoices()

	// 路由设置
	mux := http.NewServeMux()

	// API 路由
	mux.HandleFunc("/api/voices", handleVoices)
	mux.HandleFunc("/api/voices/", handleVoiceSample)
	mux.HandleFunc("/api/preview", handlePreview)
	mux.HandleFunc("/api/synthesize", handleSynthesize)

	// 静态文件
	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	// 启动服务器
	log.Printf("Edge TTS Web 服务启动: http://localhost%s", *addr)

	if *openBrowser {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openURL(fmt.Sprintf("http://localhost%s", *addr))
		}()
	}

	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}

func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}

func preloadVoices() {
	ctx := newTimeoutContext()
	voices, err := edgetts.ListVoices(ctx, nil)
	if err != nil {
		log.Printf("预加载语音列表失败: %v", err)
		return
	}
	voicesCache = voices
	voicesCacheTime = time.Now()
	log.Printf("已加载 %d 个语音", len(voices))
}

// LanguageGroup 语言分组
type LanguageGroup struct {
	Code   string       `json:"code"`
	Name   string       `json:"name"`
	Voices []VoiceInfo  `json:"voices"`
}

// VoiceInfo 语音信息
type VoiceInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Gender      string   `json:"gender"`
	Locale      string   `json:"locale"`
	Styles      []string `json:"styles"`
}

// 语言基础名称映射（语言代码 -> 语言名称）
var baseLanguageNames = map[string]string{
	"af": "南非荷兰语", "am": "阿姆哈拉语", "ar": "阿拉伯语", "az": "阿塞拜疆语",
	"bg": "保加利亚语", "bn": "孟加拉语", "bs": "波斯尼亚语", "ca": "加泰罗尼亚语",
	"cs": "捷克语", "cy": "威尔士语", "da": "丹麦语", "de": "德语",
	"el": "希腊语", "en": "英语", "es": "西班牙语", "et": "爱沙尼亚语",
	"fa": "波斯语", "fi": "芬兰语", "fil": "菲律宾语", "fr": "法语",
	"ga": "爱尔兰语", "gl": "加利西亚语", "gu": "古吉拉特语", "he": "希伯来语",
	"hi": "印地语", "hr": "克罗地亚语", "hu": "匈牙利语", "id": "印度尼西亚语",
	"is": "冰岛语", "it": "意大利语", "iu": "因纽特语", "ja": "日语",
	"jv": "爪哇语", "ka": "格鲁吉亚语", "kk": "哈萨克语", "km": "高棉语",
	"kn": "卡纳达语", "ko": "韩语", "lo": "老挝语", "lt": "立陶宛语",
	"lv": "拉脱维亚语", "mk": "马其顿语", "ml": "马拉雅拉姆语", "mn": "蒙古语",
	"mr": "马拉地语", "ms": "马来语", "mt": "马耳他语", "my": "缅甸语",
	"nb": "挪威语", "ne": "尼泊尔语", "nl": "荷兰语", "pl": "波兰语",
	"ps": "普什图语", "pt": "葡萄牙语", "ro": "罗马尼亚语", "ru": "俄语",
	"si": "僧伽罗语", "sk": "斯洛伐克语", "sl": "斯洛文尼亚语", "so": "索马里语",
	"sq": "阿尔巴尼亚语", "sr": "塞尔维亚语", "su": "巽他语", "sv": "瑞典语",
	"sw": "斯瓦希里语", "ta": "泰米尔语", "te": "泰卢固语", "th": "泰语",
	"tr": "土耳其语", "uk": "乌克兰语", "ur": "乌尔都语", "uz": "乌兹别克语",
	"vi": "越南语", "zh": "中文", "zu": "祖鲁语",
}

// 地区代码到中文名称映射
var regionNames = map[string]string{
	// 国家/地区
	"AE": "阿联酋", "AF": "阿富汗", "AL": "阿尔巴尼亚", "AR": "阿根廷",
	"AT": "奥地利", "AU": "澳大利亚", "AZ": "阿塞拜疆", "BA": "波黑",
	"BD": "孟加拉", "BE": "比利时", "BH": "巴林", "BO": "玻利维亚",
	"BR": "巴西", "CA": "加拿大", "CH": "瑞士", "CL": "智利",
	"CN": "大陆", "CO": "哥伦比亚", "CR": "哥斯达黎加", "CU": "古巴",
	"CZ": "捷克", "DE": "德国", "DK": "丹麦", "DO": "多米尼加",
	"DZ": "阿尔及利亚", "EC": "厄瓜多尔", "EE": "爱沙尼亚", "EG": "埃及",
	"ES": "西班牙", "ET": "埃塞俄比亚", "FI": "芬兰", "FR": "法国",
	"GB": "英国", "GE": "格鲁吉亚", "GQ": "赤道几内亚", "GR": "希腊",
	"GT": "危地马拉", "HK": "香港", "HN": "洪都拉斯", "HR": "克罗地亚",
	"HU": "匈牙利", "ID": "印尼", "IE": "爱尔兰", "IL": "以色列",
	"IN": "印度", "IQ": "伊拉克", "IR": "伊朗", "IS": "冰岛",
	"IT": "意大利", "JO": "约旦", "JP": "日本", "KE": "肯尼亚",
	"KH": "柬埔寨", "KR": "韩国", "KW": "科威特", "KZ": "哈萨克斯坦",
	"LA": "老挝", "LB": "黎巴嫩", "LK": "斯里兰卡", "LT": "立陶宛",
	"LV": "拉脱维亚", "LY": "利比亚", "MA": "摩洛哥", "MK": "北马其顿",
	"MM": "缅甸", "MN": "蒙古", "MT": "马耳他", "MX": "墨西哥",
	"MY": "马来西亚", "NG": "尼日利亚", "NI": "尼加拉瓜", "NL": "荷兰",
	"NO": "挪威", "NP": "尼泊尔", "NZ": "新西兰", "OM": "阿曼",
	"PA": "巴拿马", "PE": "秘鲁", "PH": "菲律宾", "PK": "巴基斯坦",
	"PL": "波兰", "PR": "波多黎各", "PT": "葡萄牙", "PY": "巴拉圭",
	"QA": "卡塔尔", "RO": "罗马尼亚", "RS": "塞尔维亚", "RU": "俄罗斯",
	"SA": "沙特", "SE": "瑞典", "SG": "新加坡", "SI": "斯洛文尼亚",
	"SK": "斯洛伐克", "SO": "索马里", "SV": "萨尔瓦多", "SY": "叙利亚",
	"TH": "泰国", "TN": "突尼斯", "TR": "土耳其", "TW": "台湾",
	"TZ": "坦桑尼亚", "UA": "乌克兰", "US": "美国", "UY": "乌拉圭",
	"UZ": "乌兹别克斯坦", "VE": "委内瑞拉", "VN": "越南", "YE": "也门",
	"ZA": "南非",
	// 特殊标识
	"Cans": "音节文字", "Latn": "拉丁文字",
}

// 中文方言映射
var chineseDialects = map[string]string{
	"liaoning": "辽宁/东北话",
	"shaanxi":  "陕西话",
}

// 特殊 locale 完整映射（优先级最高）
var specialLocales = map[string]string{
	"zh-CN":          "中文（简体）",
	"zh-TW":          "中文（繁体）",
	"zh-HK":          "中文（粤语）",
	"zh-CN-liaoning": "中文（方言·东北话）",
	"zh-CN-shaanxi":  "中文（方言·陕西话）",
	"iu-Cans-CA":     "因纽特语（音节文字）",
	"iu-Latn-CA":     "因纽特语（拉丁文字）",
}

// 语音名称本地化
var voiceNameMap = map[string]string{
	"zh-CN-XiaoxiaoNeural":     "晓晓",
	"zh-CN-YunxiNeural":        "云希",
	"zh-CN-YunjianNeural":      "云健",
	"zh-CN-XiaoyiNeural":       "晓依",
	"zh-CN-YunyangNeural":      "云扬",
	"zh-CN-XiaochenNeural":     "晓辰",
	"zh-CN-XiaohanNeural":      "晓涵",
	"zh-CN-XiaomengNeural":     "晓梦",
	"zh-CN-XiaomoNeural":       "晓墨",
	"zh-CN-XiaoqiuNeural":      "晓秋",
	"zh-CN-XiaoruiNeural":      "晓睿",
	"zh-CN-XiaoshuangNeural":   "晓双",
	"zh-CN-XiaoxuanNeural":     "晓萱",
	"zh-CN-XiaoyanNeural":      "晓颜",
	"zh-CN-XiaoyouNeural":      "晓悠",
	"zh-CN-XiaozhenNeural":     "晓甄",
	"zh-CN-YunfengNeural":      "云枫",
	"zh-CN-YunhaoNeural":       "云皓",
	"zh-CN-YunxiaNeural":       "云夏",
	"zh-CN-YunyeNeural":        "云野",
	"zh-CN-YunzeNeural":        "云泽",
	"zh-TW-HsiaoChenNeural":    "晓臻",
	"zh-TW-HsiaoYuNeural":      "晓雨",
	"zh-TW-YunJheNeural":       "云哲",
	"zh-HK-HiuGaaiNeural":      "晓佳",
	"zh-HK-HiuMaanNeural":      "晓曼",
	"zh-HK-WanLungNeural":      "云龙",
}

func getLanguageName(locale string) string {
	// 1. 优先检查特殊 locale 完整映射
	if name, ok := specialLocales[locale]; ok {
		return name
	}

	parts := strings.Split(locale, "-")
	if len(parts) < 2 {
		// 只有语言代码
		if name, ok := baseLanguageNames[locale]; ok {
			return name
		}
		return locale
	}

	langCode := parts[0]
	regionCode := parts[1]

	// 2. 获取基础语言名称
	langName, hasLang := baseLanguageNames[langCode]
	if !hasLang {
		return locale
	}

	// 3. 获取地区名称
	regionName, hasRegion := regionNames[regionCode]
	if !hasRegion {
		regionName = regionCode
	}

	// 4. 只有一个地区变体的语言，不显示地区后缀
	singleRegionLangs := map[string]bool{
		"af": true, "am": true, "az": true, "bg": true, "bs": true, "ca": true,
		"cs": true, "cy": true, "da": true, "el": true, "et": true, "fa": true,
		"fi": true, "fil": true, "ga": true, "gl": true, "gu": true, "he": true,
		"hi": true, "hr": true, "hu": true, "id": true, "is": true, "ja": true,
		"jv": true, "ka": true, "kk": true, "km": true, "kn": true, "ko": true,
		"lo": true, "lt": true, "lv": true, "mk": true, "ml": true, "mn": true,
		"mr": true, "ms": true, "mt": true, "my": true, "nb": true, "ne": true,
		"pl": true, "ps": true, "ro": true, "ru": true, "si": true, "sk": true,
		"sl": true, "so": true, "sq": true, "sr": true, "su": true, "sv": true,
		"te": true, "th": true, "tr": true, "uk": true, "uz": true, "vi": true,
		"zu": true,
	}

	if singleRegionLangs[langCode] {
		return langName
	}

	return langName + "（" + regionName + "）"
}

func getVoiceDisplayName(shortName string, originalName string) string {
	if name, ok := voiceNameMap[shortName]; ok {
		return name
	}
	// 从 FriendlyName 提取
	// 格式如: "Microsoft Xiaoxiao Online (Natural) - Chinese (Mainland)"
	parts := strings.Split(originalName, " ")
	if len(parts) >= 2 {
		return parts[1]
	}
	return shortName
}

func handleVoices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 使用缓存或重新获取
	if len(voicesCache) == 0 || time.Since(voicesCacheTime) > 30*time.Minute {
		preloadVoices()
	}

	// 按语言分组
	groups := make(map[string]*LanguageGroup)

	for _, v := range voicesCache {
		locale := v.Locale

		if _, exists := groups[locale]; !exists {
			groups[locale] = &LanguageGroup{
				Code:   locale,
				Name:   getLanguageName(locale),
				Voices: []VoiceInfo{},
			}
		}

		groups[locale].Voices = append(groups[locale].Voices, VoiceInfo{
			ID:     v.ShortName,
			Name:   getVoiceDisplayName(v.ShortName, v.FriendlyName),
			Gender: v.Gender,
			Locale: v.Locale,
			Styles: v.VoiceTag.VoicePersonalities,
		})
	}

	// 转换为数组并排序（按语言分组，中文优先，英文其次）
	result := make([]LanguageGroup, 0, len(groups))

	// 收集所有 locale
	allLocales := make([]string, 0, len(groups))
	for locale := range groups {
		allLocales = append(allLocales, locale)
	}

	// 获取语言前缀（如 zh-CN -> zh, en-US -> en）
	getLangPrefix := func(locale string) string {
		if idx := strings.Index(locale, "-"); idx > 0 {
			return locale[:idx]
		}
		return locale
	}

	// 排序优先级：zh > en > ja > ko > 其他（按字母顺序）
	langPriority := map[string]int{
		"zh": 0,
		"en": 1,
		"ja": 2,
		"ko": 3,
	}

	// 排序函数
	sort.Slice(allLocales, func(i, j int) bool {
		prefixI := getLangPrefix(allLocales[i])
		prefixJ := getLangPrefix(allLocales[j])

		// 比较语言优先级
		priI, okI := langPriority[prefixI]
		priJ, okJ := langPriority[prefixJ]

		if okI && okJ {
			// 都是优先语言，按优先级排序
			if priI != priJ {
				return priI < priJ
			}
		} else if okI {
			// 只有 i 是优先语言
			return true
		} else if okJ {
			// 只有 j 是优先语言
			return false
		}

		// 同一语言内部或其他语言：先按语言前缀排序，再按完整 locale 排序
		if prefixI != prefixJ {
			return prefixI < prefixJ
		}
		return allLocales[i] < allLocales[j]
	})

	// 按排序后的顺序添加
	for _, locale := range allLocales {
		result = append(result, *groups[locale])
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"languages": result,
	})
}

func handleVoiceSample(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从 URL 提取 voice ID
	// /api/voices/{id}/sample
	path := strings.TrimPrefix(r.URL.Path, "/api/voices/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "sample" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	voiceID := parts[0]

	// 根据语言选择示例文本
	sampleText := getSampleText(voiceID)

	ctx := newTimeoutContext()
	comm, err := edgetts.NewCommunicate(sampleText, voiceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	if err := comm.StreamToWriter(ctx, w, nil); err != nil {
		log.Printf("语音合成错误: %v", err)
	}
}

func getSampleText(voiceID string) string {
	if strings.HasPrefix(voiceID, "zh-CN") {
		return "你好，我是智能语音助手，很高兴为您服务。"
	} else if strings.HasPrefix(voiceID, "zh-TW") {
		return "你好，我是智慧語音助手，很高興為您服務。"
	} else if strings.HasPrefix(voiceID, "zh-HK") {
		return "你好，我係智能語音助手，好高興為你服務。"
	} else if strings.HasPrefix(voiceID, "ja-JP") {
		return "こんにちは、私は音声アシスタントです。"
	} else if strings.HasPrefix(voiceID, "ko-KR") {
		return "안녕하세요, 저는 음성 도우미입니다."
	} else if strings.HasPrefix(voiceID, "en") {
		return "Hello, I am an AI voice assistant. Nice to meet you."
	} else if strings.HasPrefix(voiceID, "fr") {
		return "Bonjour, je suis un assistant vocal intelligent."
	} else if strings.HasPrefix(voiceID, "de") {
		return "Hallo, ich bin ein intelligenter Sprachassistent."
	} else if strings.HasPrefix(voiceID, "es") {
		return "Hola, soy un asistente de voz inteligente."
	} else if strings.HasPrefix(voiceID, "ru") {
		return "Привет, я голосовой помощник."
	}
	return "Hello, I am a voice assistant."
}

// PreviewRequest 预览请求
type PreviewRequest struct {
	Text   string `json:"text"`
	Voice  string `json:"voice"`
	Rate   string `json:"rate"`
	Pitch  string `json:"pitch"`
}

func handlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return
	}

	if req.Voice == "" {
		req.Voice = edgetts.DefaultVoice
	}
	if req.Rate == "" {
		req.Rate = "+0%"
	}
	if req.Pitch == "" {
		req.Pitch = "+0Hz"
	}

	ctx := newTimeoutContext()
	comm, err := edgetts.NewCommunicate(
		req.Text,
		req.Voice,
		edgetts.WithRate(req.Rate),
		edgetts.WithPitch(req.Pitch),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "audio/mpeg")

	if err := comm.StreamToWriter(ctx, w, nil); err != nil {
		log.Printf("预览合成错误: %v", err)
	}
}

// SynthesizeRequest 合成请求
type SynthesizeRequest struct {
	Text        string `json:"text"`
	Voice       string `json:"voice"`
	Rate        string `json:"rate"`
	Pitch       string `json:"pitch"`
	WithSRT     bool   `json:"withSrt"`
}

func handleSynthesize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SynthesizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return
	}

	if req.Voice == "" {
		req.Voice = edgetts.DefaultVoice
	}
	if req.Rate == "" {
		req.Rate = "+0%"
	}
	if req.Pitch == "" {
		req.Pitch = "+0Hz"
	}

	ctx := newTimeoutContext()
	comm, err := edgetts.NewCommunicate(
		req.Text,
		req.Voice,
		edgetts.WithRate(req.Rate),
		edgetts.WithPitch(req.Pitch),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.WithSRT {
		// 返回 JSON，包含音频的 base64 和 SRT
		handleSynthesizeWithSRT(w, ctx, comm)
	} else {
		// 直接返回音频流
		filename := fmt.Sprintf("tts_%d.mp3", time.Now().Unix())
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

		if err := comm.StreamToWriter(ctx, w, nil); err != nil {
			log.Printf("合成错误: %v", err)
		}
	}
}

func handleSynthesizeWithSRT(w http.ResponseWriter, ctx contextWithTimeout, comm *edgetts.Communicate) {
	submaker := edgetts.NewSubMaker()

	// 收集音频数据
	var audioData []byte

	chunkCh, errCh := comm.Stream(ctx)

	for {
		select {
		case chunk, ok := <-chunkCh:
			if !ok {
				goto done
			}
			if chunk.Type == "audio" {
				audioData = append(audioData, chunk.Data...)
			} else if chunk.Type == "WordBoundary" || chunk.Type == "SentenceBoundary" {
				submaker.Feed(chunk)
			}
		case err := <-errCh:
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case <-ctx.Done():
			http.Error(w, "Timeout", http.StatusRequestTimeout)
			return
		}
	}

done:
	// 编码为 base64
	audioBase64 := base64.StdEncoding.EncodeToString(audioData)
	srtContent := submaker.GetSRT()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"audio": audioBase64,
		"srt":   srtContent,
	})
}

type contextWithTimeout = context.Context

func newTimeoutContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Minute)
	return ctx
}
