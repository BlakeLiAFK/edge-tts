package edgetts

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

var multiWSRegex = regexp.MustCompile(`\n\n+`)

// Subtitle 字幕条目
type Subtitle struct {
	Index   int
	Start   time.Duration
	End     time.Duration
	Content string
}

// ToSRT 将字幕转换为 SRT 格式块
func (s *Subtitle) ToSRT(eol string) string {
	if eol == "" {
		eol = "\n"
	}

	content := makeLegalContent(s.Content)
	if eol != "\n" {
		content = strings.ReplaceAll(content, "\n", eol)
	}

	idx := s.Index
	if idx == 0 {
		idx = 0
	}

	return fmt.Sprintf("%d%s%s --> %s%s%s%s%s",
		idx, eol,
		timeDurationToSRTTimestamp(s.Start), timeDurationToSRTTimestamp(s.End), eol,
		content, eol, eol)
}

// makeLegalContent 移除非法内容
func makeLegalContent(content string) string {
	if content != "" && content[0] != '\n' && !strings.Contains(content, "\n\n") {
		return content
	}

	content = strings.Trim(content, "\n")
	return multiWSRegex.ReplaceAllString(content, "\n")
}

// timeDurationToSRTTimestamp 将 time.Duration 转换为 SRT 时间戳
func timeDurationToSRTTimestamp(d time.Duration) string {
	totalSeconds := int64(d.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	milliseconds := (d.Milliseconds()) % 1000

	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, milliseconds)
}

// sortAndReindex 排序并重新索引字幕
func sortAndReindex(subtitles []Subtitle, startIndex int, skip bool) []Subtitle {
	// 复制并排序
	sorted := make([]Subtitle, len(subtitles))
	copy(sorted, subtitles)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Start != sorted[j].Start {
			return sorted[i].Start < sorted[j].Start
		}
		if sorted[i].End != sorted[j].End {
			return sorted[i].End < sorted[j].End
		}
		return sorted[i].Index < sorted[j].Index
	})

	var result []Subtitle
	idx := startIndex
	skipped := 0

	for _, sub := range sorted {
		if skip {
			// 检查是否应该跳过
			if strings.TrimSpace(sub.Content) == "" {
				skipped++
				continue
			}
			if sub.Start < 0 {
				skipped++
				continue
			}
			if sub.Start >= sub.End {
				skipped++
				continue
			}
		}

		newSub := Subtitle{
			Index:   idx - skipped,
			Start:   sub.Start,
			End:     sub.End,
			Content: sub.Content,
		}
		result = append(result, newSub)
		idx++
	}

	return result
}

// ComposeSRT 组合字幕为 SRT 字符串
func ComposeSRT(subtitles []Subtitle, reindex bool, startIndex int, eol string) string {
	if reindex {
		subtitles = sortAndReindex(subtitles, startIndex, true)
	}

	var builder strings.Builder
	for _, sub := range subtitles {
		builder.WriteString(sub.ToSRT(eol))
	}
	return builder.String()
}
