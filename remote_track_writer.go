package main

import (
	"errors"

	"github.com/livekit/media-sdk"
	"go.uber.org/atomic"
)

var ErrClosed = errors.New("writer is closed")

type RemoteTrackWriter struct {
	handler *ASRAPIHandler
	closed  atomic.Bool
}

func NewRemoteTrackWriter(handler *ASRAPIHandler) *RemoteTrackWriter {
	return &RemoteTrackWriter{
		handler: handler,
	}
}

func (w *RemoteTrackWriter) WriteSample(sample media.PCM16Sample) error {
	if w.closed.Load() {
		return ErrClosed
	}

	// 会议室音频包写入asr接口
	return w.handler.SendAudioChunk(sample)
}

func (w *RemoteTrackWriter) Close() error {
	w.closed.Swap(true)
	return nil
}
