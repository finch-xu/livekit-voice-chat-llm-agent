package main

import (
	"encoding/binary"
	"github.com/livekit/media-sdk"
	"os"
	"os/signal"
	"syscall"

	"github.com/livekit/protocol/logger"
	lksdk "github.com/livekit/server-sdk-go/v2"
	lkmedia "github.com/livekit/server-sdk-go/v2/pkg/media"
	"github.com/pion/webrtc/v4"
)

func callbacksForLkRoom(handler *ASRAPIHandler) *lksdk.RoomCallback {
	// livekit的房间回调函数

	var pcmRemoteTrack *lkmedia.PCMRemoteTrack

	return &lksdk.RoomCallback{
		ParticipantCallback: lksdk.ParticipantCallback{
			OnTrackSubscribed: func(track *webrtc.TrackRemote, publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
				if pcmRemoteTrack != nil {
					// only handle one track
					return
				}
				pcmRemoteTrack, _ = handleSubscribe(track, handler)
			},
		},
		OnDisconnected: func() {
			if pcmRemoteTrack != nil {
				pcmRemoteTrack.Close()
				pcmRemoteTrack = nil
			}
		},
		OnDisconnectedWithReason: func(reason lksdk.DisconnectionReason) {
			if pcmRemoteTrack != nil {
				pcmRemoteTrack.Close()
				pcmRemoteTrack = nil
			}
		},
	}
}

func main() {
	// 读取env配置
	//loadEnv()

	//// livekit房间音频流推送进asr ws接口
	//asrAudioChan := make(chan media.PCM16Sample)
	//defer close(asrAudioChan)

	// tts ws的音频流发送到livekit的管道
	ttsAudioChan := make(chan []byte)
	defer close(ttsAudioChan)

	// livekit文本消息推送管道
	dataStreamChan := make(chan string)
	defer close(dataStreamChan)

	// asr文本结果推送到llm
	asrToLLMChan := make(chan string)
	defer close(asrToLLMChan)

	// llm sse结果流式推送到tts ws接口
	llmToTTSChan := make(chan string)
	defer close(llmToTTSChan)

	// asr主程序
	handler, err := NewASRAPIHandler(asrToLLMChan)
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	// 大模型主程序
	go llm(asrToLLMChan, llmToTTSChan)

	// 创建并连接livekit房间
	room, err := connectToLKRoom(callbacksForLkRoom(handler))
	if err != nil {
		panic(err)
	}
	defer room.Disconnect()

	//go datastream(room, dataStreamChan)

	// tts主程序
	NewTTSAPIHandler(llmToTTSChan, ttsAudioChan)

	// 音频推送goroutine
	go handlePublish(room, ttsAudioChan)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	<-sigChan
}

func handlePublish(room *lksdk.Room, ttsAudioChan chan []byte) {
	// 创建一个本地PCM轨道，采样率16khz，单声道，16bit小端序，用来匹配接收TTS音频流
	publishTrack, err := lkmedia.NewPCMLocalTrack(16000, 1, logger.GetLogger())
	if err != nil {
		panic(err)
	}
	defer func() {
		publishTrack.ClearQueue()
		publishTrack.Close()
	}()
	// 创建livekit的本地音频推送轨道，sdk会自动把本地PCM轨道转成本地Opus轨道，这样livekit就能接收了
	if _, err = room.LocalParticipant.PublishTrack(publishTrack, &lksdk.TrackPublicationOptions{
		Name: "tts audio",
	}); err != nil {
		panic(err)
	}
	// 接收tts发过来的原始PCM音频包
	for sample := range ttsAudioChan {
		// 把tts音频包转成PCM16Sample包
		//audioBase64 := string(sample)
		//audioBytes, err := base64.StdEncoding.DecodeString(audioBase64)
		//if err != nil {
		//	panic(err)
		//}
		audioPCM16 := make(media.PCM16Sample, len(sample)/2)
		for i := 0; i < len(sample); i += 2 {
			// 转成小端序pcm包
			audioPCM16[i/2] = int16(binary.LittleEndian.Uint16(sample[i : i+2]))
		}
		// 发送音频流到livekit会议室
		if err := publishTrack.WriteSample(audioPCM16); err != nil {
			logger.Errorw("Failed to write sample", err)
		}
	}
}

func handleSubscribe(track *webrtc.TrackRemote, handler *ASRAPIHandler) (*lkmedia.PCMRemoteTrack, error) {
	if track.Codec().MimeType != webrtc.MimeTypeOpus {
		logger.Warnw("Received non-opus track", nil, "track", track.Codec().MimeType)
	}

	writer := NewRemoteTrackWriter(handler)
	trackWriter, err := lkmedia.NewPCMRemoteTrack(track, writer, lkmedia.WithTargetSampleRate(16000))
	if err != nil {
		logger.Errorw("Failed to create remote track", err)
		return nil, err
	}

	return trackWriter, nil
}
