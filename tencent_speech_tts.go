package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	//
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/livekit/protocol/logger"
)

type TTSAPIHandler struct {
	conn  *websocket.Conn
	url   string
	inCh  chan string
	outCh chan []byte
}

func NewTTSAPIHandler(llmToTTSChan chan string, ttsAudioChan chan []byte) (*TTSAPIHandler, error) {
	wsUrl := "wss://tts.cloud.tencent.com/stream_wsv2?"

	timestamp2 := time.Now().Unix()             // 当前时间戳
	expired := timestamp2 + (30 * 24 * 60 * 60) // 当前时间 + 30 天有效期

	appId, _ := strconv.Atoi(appId)

	sessionId := uuid.New().String()

	var requestBody = map[string]string{
		"Action":          "TextToStreamAudioWSv2",
		"AppId":           strconv.Itoa(appId),
		"SecretId":        secretId,
		"timestamp":       strconv.FormatInt(time.Now().Unix(), 10),
		"expired":         strconv.FormatInt(expired, 10),
		"SessionId":       sessionId,
		"VoiceType":       strconv.Itoa(502001), // 音色id，参考腾讯tts音色列表
		"Codec":           "pcm",                // 音频编码格式
		"SampleRate":      strconv.Itoa(16000),  // 采样率
		"EmotionCategory": "neutral",            // 情感
	}

	// 构造签名
	keys := make([]string, 0, len(requestBody))
	for key := range requestBody {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var result strings.Builder
	for i, key := range keys {
		result.WriteString(url.QueryEscape(key))
		result.WriteString("=")
		result.WriteString(url.QueryEscape(requestBody[key]))
		if i < len(keys)-1 {
			result.WriteString("&")
		}
	}
	originRequestString := result.String()
	originSignString := "GET" + "tts.cloud.tencent.com/stream_wsv2?" + originRequestString

	h := hmac.New(sha1.New, []byte(secretKey))
	h.Write([]byte(originSignString))
	hmacResult := h.Sum(nil)
	base64Result := base64.StdEncoding.EncodeToString(hmacResult)
	urlEncodedSignature := url.QueryEscape(base64Result)
	wsUrl = wsUrl + originRequestString + "&Signature=" + urlEncodedSignature

	// 建立 WebSocket 连接
	dialer := &websocket.Dialer{}
	conn, _, err := dialer.Dial(wsUrl, nil)
	if err != nil {
		return nil, err
	}

	tts := &TTSAPIHandler{
		conn:  conn,
		url:   wsUrl,
		inCh:  llmToTTSChan,
		outCh: ttsAudioChan,
	}

	go tts.sendText()
	go tts.readMessages()

	return tts, nil
}

// 把大模型推送的文本发送到TTS接口
func (h *TTSAPIHandler) sendText() error {
	defer func() {
		h.Close()
	}()
	for {
		select {
		case text := <-h.inCh:
			// 构造tts消息
			body := map[string]interface{}{
				"action": "ACTION_SYNTHESIS",
				"data":   text,
			}
			bytes, _ := json.Marshal(body)
			err := h.conn.WriteMessage(websocket.TextMessage, bytes)
			if err != nil {
				return err
			}
		}
	}
}

// 接收TTS接口返回的音频流
func (h *TTSAPIHandler) readMessages() error {
	defer func() {
		h.Close()
	}()
	for {
		messageType, message, err := h.conn.ReadMessage()
		if err != nil {
			logger.Errorw("Error reading message", err)
			return err
		}

		if messageType == websocket.BinaryMessage {
			h.outCh <- message
		}
	}
}

func (m *TTSAPIHandler) Close() error {
	return m.conn.Close()
}
