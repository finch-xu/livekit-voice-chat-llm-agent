# livekit-voice-chat-llm-agent

webrtc livekit voice chat llm Agent, include ASR TTS LLM

开发一个对接到livekit会议室的语音对话机器人，全程ASR实时流式转写、LLM流式问答、TTS实时流式播报。

ASR、TTS对接腾讯语音接口，llm使用的openai兼容格式，百度千帆大模型的免费接口。

整体流程的文档写到了 https://pidan.dev/20250719/webrtc-connect-livekit-use-go-sdk/

非常感谢livekit官方的努力，让一切变得这么简单。        
本项目基于 https://github.com/livekit/server-sdk-go/tree/main/examples/openai_realtime_voice 修改并扩展开发。

设置go源

```bash
go env -w GOPROXY=https://goproxy.cn,direct    
```

安装opus依赖

```bash
apt-get install -y pkg-config libopus-dev libopusfile-dev libsoxr-dev
```

https://github.com/hraban/opus

```bash
go mod tidy
go build -tags cgo -o main
```









