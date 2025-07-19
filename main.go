package main

import (
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

	//// tts ws的音频流发送到livekit的管道
	//ttsAudioChan := make(chan media.PCM16Sample)
	//defer close(ttsAudioChan)
	//
	// livekit文本消息推送管道
	dataStreamChan := make(chan string)
	defer close(dataStreamChan)
	//
	// asr文本结果推送到llm
	asrToLLMChan := make(chan string)
	defer close(asrToLLMChan)
	//
	//// llm sse结果流式推送到tts ws接口
	//llmToTTSChan := make(chan string)
	//defer close(llmToTTSChan)

	handler, err := NewASRAPIHandler(asrToLLMChan)
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	go llm(asrToLLMChan, dataStreamChan)

	room, err := connectToLKRoom(callbacksForLkRoom(handler))
	if err != nil {
		panic(err)
	}
	defer room.Disconnect()

	go datastream(room, dataStreamChan)

	//go handlePublish(room, ttsAudioChan)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	<-sigChan
}

//func handlePublish(room *lksdk.Room, audioWriterChan chan media.PCM16Sample) {
//	publishTrack, err := lkmedia.NewPCMLocalTrack(16000, 1, logger.GetLogger())
//	if err != nil {
//		panic(err)
//	}
//	defer func() {
//		publishTrack.ClearQueue()
//		publishTrack.Close()
//	}()
//
//	if _, err = room.LocalParticipant.PublishTrack(publishTrack, &lksdk.TrackPublicationOptions{
//		Name: "tts audio",
//	}); err != nil {
//		panic(err)
//	}
//
//	for sample := range audioWriterChan {
//		if err := publishTrack.WriteSample(sample); err != nil {
//			logger.Errorw("Failed to write sample", err)
//		}
//	}
//}

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
