# Edge TTS Go

基于 Microsoft Edge 在线文本转语音服务的 Go 语言实现。

## 特性

- 支持 400+ 种语音，覆盖 100+ 种语言和地区
- 支持调节语速、音量、音调
- 支持生成 SRT 字幕文件
- 提供命令行工具和 Web 界面
- 支持流式输出
- 纯 Go 实现，无需外部依赖

## 安装

### 从源码编译

```bash
# 克隆仓库
git clone https://github.com/BlakeLiAFK/edge-tts.git
cd edge-tts

# 编译命令行工具
go build -o edge-tts ./cmd/edge-tts

# 编译 Web 服务
go build -o edge-tts-web ./cmd/edge-tts-web
```

### 使用 Go Install

```bash
# 安装命令行工具
go install github.com/BlakeLiAFK/edge-tts/cmd/edge-tts@latest

# 安装 Web 服务
go install github.com/BlakeLiAFK/edge-tts/cmd/edge-tts-web@latest
```

## 使用方法

### 命令行工具

```bash
# 基本使用
edge-tts -t "你好，世界" -o output.mp3

# 指定语音
edge-tts -t "Hello World" -v en-US-AriaNeural -o output.mp3

# 调节语速和音量
edge-tts -t "快速播放" -r "+50%" -vol "+20%" -o output.mp3

# 生成字幕文件
edge-tts -t "生成字幕测试" -o output.mp3 -s output.srt

# 列出所有可用语音
edge-tts -l
```

#### 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-t` | 要转换的文本 | - |
| `-v` | 语音名称 | zh-CN-XiaoxiaoNeural |
| `-o` | 输出音频文件 | output.mp3 |
| `-s` | 输出字幕文件 | - |
| `-r` | 语速 (-100% ~ +100%) | +0% |
| `-vol` | 音量 (-100% ~ +100%) | +0% |
| `-p` | 音调 (-100Hz ~ +100Hz) | +0Hz |
| `-l` | 列出所有可用语音 | - |
| `-proxy` | 代理服务器地址 | - |

### Web 服务

```bash
# 启动服务（默认端口 8080）
edge-tts-web

# 指定端口
edge-tts-web -addr :9000

# 启动后自动打开浏览器
edge-tts-web -open
```

启动后访问 http://localhost:8080 即可使用 Web 界面。

#### Web 界面功能

- 支持选择语言和语音
- 实时预览语音效果
- 调节语速、音量、音调
- 下载 MP3 音频文件
- 可选生成 SRT 字幕
- 历史记录保存

### 作为库使用

```go
package main

import (
    "context"
    "log"

    "github.com/BlakeLiAFK/edge-tts/pkg/edgetts"
)

func main() {
    // 创建通信实例
    comm, err := edgetts.NewCommunicate(
        "你好，这是一段测试文本",
        "zh-CN-XiaoxiaoNeural",
        edgetts.WithRate("+10%"),
        edgetts.WithVolume("+0%"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // 保存到文件
    err = comm.Save(context.Background(), "output.mp3", "output.srt")
    if err != nil {
        log.Fatal(err)
    }
}
```

#### 流式处理

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/BlakeLiAFK/edge-tts/pkg/edgetts"
)

func main() {
    comm, _ := edgetts.NewCommunicate("Hello World", "en-US-AriaNeural")

    chunkCh, errCh := comm.Stream(context.Background())

    file, _ := os.Create("output.mp3")
    defer file.Close()

    for {
        select {
        case chunk, ok := <-chunkCh:
            if !ok {
                return
            }
            if chunk.Type == "audio" {
                file.Write(chunk.Data)
            } else {
                fmt.Printf("字幕: %s (offset: %d)\n", chunk.Text, chunk.Offset)
            }
        case err := <-errCh:
            if err != nil {
                panic(err)
            }
        }
    }
}
```

## 可用语音

支持以下语言和地区的语音（部分列表）：

| 语言 | 语音示例 |
|------|----------|
| 中文（简体） | zh-CN-XiaoxiaoNeural, zh-CN-YunxiNeural |
| 中文（粤语） | zh-HK-HiuGaaiNeural, zh-HK-WanLungNeural |
| 中文（繁体） | zh-TW-HsiaoChenNeural, zh-TW-YunJheNeural |
| 英语（美国） | en-US-AriaNeural, en-US-GuyNeural |
| 英语（英国） | en-GB-SoniaNeural, en-GB-RyanNeural |
| 日语 | ja-JP-NanamiNeural, ja-JP-KeitaNeural |
| 韩语 | ko-KR-SunHiNeural, ko-KR-InJoonNeural |

使用 `edge-tts -l` 查看完整语音列表。

## API 接口

Web 服务提供以下 REST API：

### 获取语音列表

```
GET /api/voices
```

响应示例：
```json
{
  "languages": [
    {
      "code": "zh-CN",
      "name": "中文（简体）",
      "voices": [
        {
          "id": "zh-CN-XiaoxiaoNeural",
          "name": "晓晓",
          "gender": "Female"
        }
      ]
    }
  ]
}
```

### 合成语音

```
POST /api/synthesize
Content-Type: application/json

{
  "text": "要转换的文本",
  "voice": "zh-CN-XiaoxiaoNeural",
  "rate": "+0%",
  "volume": "+0%",
  "pitch": "+0Hz",
  "subtitle": false
}
```

响应：音频文件流（audio/mpeg）

### 预览语音

```
GET /api/voices/{voiceId}/sample
```

响应：语音预览音频流

## 项目结构

```
edge-tts/
├── cmd/
│   ├── edge-tts/          # 命令行工具
│   │   └── main.go
│   └── edge-tts-web/      # Web 服务
│       ├── main.go
│       └── static/        # 静态资源
│           ├── index.html
│           ├── css/
│           └── js/
├── pkg/
│   └── edgetts/           # 核心库
│       ├── communicate.go # 通信处理
│       ├── constants.go   # 常量定义
│       ├── drm.go         # DRM 处理
│       ├── exceptions.go  # 错误定义
│       ├── srt.go         # SRT 字幕
│       ├── submaker.go    # 字幕生成
│       ├── types.go       # 类型定义
│       ├── util.go        # 工具函数
│       └── voices.go      # 语音管理
├── go.mod
├── go.sum
└── README.md
```

## 系统要求

- Go 1.21+
- 网络连接（需要访问 Microsoft Edge TTS 服务）

## 依赖

- [gorilla/websocket](https://github.com/gorilla/websocket) - WebSocket 客户端

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！
