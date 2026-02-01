package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/BlakeLiAFK/edge-tts/pkg/edgetts"
)

const version = "1.0.0"

func printVoices(ctx context.Context, proxy string) error {
	voices, err := edgetts.ListVoices(ctx, &edgetts.ListVoicesOptions{Proxy: proxy})
	if err != nil {
		return err
	}

	// 按 ShortName 排序
	sort.Slice(voices, func(i, j int) bool {
		return voices[i].ShortName < voices[j].ShortName
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Name\tGender\tContentCategories\tVoicePersonalities")

	for _, voice := range voices {
		categories := strings.Join(voice.VoiceTag.ContentCategories, ", ")
		personalities := strings.Join(voice.VoiceTag.VoicePersonalities, ", ")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", voice.ShortName, voice.Gender, categories, personalities)
	}

	return w.Flush()
}

func runTTS(ctx context.Context, text, voice, rate, volume, pitch, proxy, writeMedia, writeSubtitles string) error {
	comm, err := edgetts.NewCommunicate(
		text,
		voice,
		edgetts.WithRate(rate),
		edgetts.WithVolume(volume),
		edgetts.WithPitch(pitch),
		edgetts.WithProxy(proxy),
	)
	if err != nil {
		return err
	}

	submaker := edgetts.NewSubMaker()

	// 确定音频输出
	var audioWriter io.Writer
	var audioFile *os.File

	if writeMedia != "" && writeMedia != "-" {
		audioFile, err = os.Create(writeMedia)
		if err != nil {
			return err
		}
		defer audioFile.Close()
		audioWriter = audioFile
	} else {
		audioWriter = os.Stdout
	}

	// 执行 TTS
	if err := comm.StreamToWriter(ctx, audioWriter, submaker); err != nil {
		return err
	}

	// 写入字幕
	if writeSubtitles != "" {
		srt := submaker.GetSRT()
		if writeSubtitles == "-" {
			fmt.Fprint(os.Stderr, srt)
		} else {
			if err := os.WriteFile(writeSubtitles, []byte(srt), 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	// 定义命令行参数
	text := flag.String("t", "", "Text to speak")
	textAlias := flag.String("text", "", "Text to speak (alias for -t)")
	file := flag.String("f", "", "Read text from file")
	fileAlias := flag.String("file", "", "Read text from file (alias for -f)")
	voice := flag.String("v", edgetts.DefaultVoice, "Voice to use")
	voiceAlias := flag.String("voice", edgetts.DefaultVoice, "Voice to use (alias for -v)")
	listVoices := flag.Bool("l", false, "List available voices")
	listVoicesAlias := flag.Bool("list-voices", false, "List available voices (alias for -l)")
	rate := flag.String("rate", "+0%", "Speech rate")
	volume := flag.String("volume", "+0%", "Speech volume")
	pitch := flag.String("pitch", "+0Hz", "Speech pitch")
	writeMedia := flag.String("write-media", "", "Output audio file")
	writeSubtitles := flag.String("write-subtitles", "", "Output subtitles file")
	proxy := flag.String("proxy", "", "Proxy URL")
	showVersion := flag.Bool("version", false, "Show version")

	flag.Parse()

	// 处理版本
	if *showVersion {
		fmt.Printf("edge-tts-go %s\n", version)
		return
	}

	ctx := context.Background()

	// 处理列出语音
	if *listVoices || *listVoicesAlias {
		if err := printVoices(ctx, *proxy); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 获取文本
	inputText := *text
	if inputText == "" {
		inputText = *textAlias
	}

	inputFile := *file
	if inputFile == "" {
		inputFile = *fileAlias
	}

	// 从文件读取
	if inputFile != "" {
		if inputFile == "-" || inputFile == "/dev/stdin" {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
				os.Exit(1)
			}
			inputText = string(data)
		} else {
			data, err := os.ReadFile(inputFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
				os.Exit(1)
			}
			inputText = string(data)
		}
	}

	if inputText == "" {
		fmt.Fprintln(os.Stderr, "Error: no text provided. Use -t or -f to specify text.")
		flag.Usage()
		os.Exit(1)
	}

	// 获取语音
	selectedVoice := *voice
	if *voiceAlias != edgetts.DefaultVoice {
		selectedVoice = *voiceAlias
	}

	// 运行 TTS
	if err := runTTS(ctx, inputText, selectedVoice, *rate, *volume, *pitch, *proxy, *writeMedia, *writeSubtitles); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
