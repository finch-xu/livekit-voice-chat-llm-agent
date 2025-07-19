package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	//
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/livekit/media-sdk"
	"github.com/livekit/protocol/logger"
)

// https://cloud.tencent.com/document/product/1093/48982

type respBody struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	VoiceId   string `json:"voice_id"`
	MessageId string `json:"message_id"`
	Final     int    `json:"final"`
	Result    struct {
		Index        int    `json:"index"`
		VoiceTextStr string `json:"voice_text_str"`
		SliceType    int    `json:"slice_type"`
	} `json:"result"`
}

type ASRAPIHandler struct {
	conn *websocket.Conn
	url  string
	ch   chan string
}

func NewASRAPIHandler(asrToLLMChan chan string) (*ASRAPIHandler, error) {

	wsUrl := "wss://asr.cloud.tencent.com/asr/v2/"

	timestamp2 := time.Now().Unix()             // 当前时间戳
	expired := timestamp2 + (30 * 24 * 60 * 60) // 当前时间 + 30 天有效期

	var requestBody = map[string]string{
		"secretid":          secretId,
		"timestamp":         strconv.FormatInt(time.Now().Unix(), 10),
		"expired":           strconv.FormatInt(expired, 10),
		"nonce":             strconv.Itoa(11111111),
		"engine_model_type": "16k_zh",            //模型类型
		"voice_id":          uuid.New().String(), // 每new一个asr ws client，这个id就必须变新的
		"voice_format":      strconv.Itoa(1),     // 输入PCM
		"needvad":           strconv.Itoa(1),
		"vad_silence_time":  strconv.Itoa(1000), // 断句的间隔
	}

	// 1. 提取所有的键
	keys := make([]string, 0, len(requestBody))
	for key := range requestBody {
		keys = append(keys, key)
	}
	// 2. 按字典序排序键
	sort.Strings(keys)
	// 3. 按排序后的顺序拼接键值对
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
	originSignString := "asr.cloud.tencent.com/asr/v2/" + appId + "?" + originRequestString

	fmt.Printf("originSignString: %s\n", originSignString)

	// 1. 使用 HMAC-SHA1 加密签名原文
	h := hmac.New(sha1.New, []byte(secretKey))
	h.Write([]byte(originSignString))
	hmacResult := h.Sum(nil)

	// 2. 对加密结果进行 Base64 编码
	base64Result := base64.StdEncoding.EncodeToString(hmacResult)

	// 3. 输出最终结果
	//fmt.Println("HMAC-SHA1 加密后 Base64 编码结果:", base64Result)
	urlEncodedSignature := url.QueryEscape(base64Result)
	// 拼接实际的请求url
	newWsUrl := wsUrl + appId + "?" + originRequestString + "&signature=" + urlEncodedSignature
	fmt.Printf("newWsUrl: %s\n", newWsUrl)

	// 请求
	dialer := &websocket.Dialer{
		//TLSClientConfig: &tls.Config{
		//	InsecureSkipVerify: true, // 忽略证书验证
		//},
	}
	conn, _, err := dialer.Dial(newWsUrl, nil)

	if err != nil {
		return nil, err
	}

	asr := &ASRAPIHandler{
		conn: conn,
		url:  newWsUrl,
		ch:   asrToLLMChan,
	}

	go asr.readMessages()
	return asr, nil
}

func (h *ASRAPIHandler) Url() string {
	return h.url
}

func (h *ASRAPIHandler) SendAudioChunk(sample media.PCM16Sample) error {
	bytes := make([]byte, len(sample)*2)
	for i, s := range sample {
		binary.LittleEndian.PutUint16(bytes[i*2:], uint16(s))
	}
	return h.conn.WriteMessage(websocket.BinaryMessage, bytes)
}

func (h *ASRAPIHandler) readMessages() {
	defer func() {
		h.Close()
	}()
	for {
		_, message, err := h.conn.ReadMessage()
		if err != nil {
			logger.Errorw("Error reading message", err)
			break
		}

		fmt.Printf("asr resp text message: %v\n", string(message))

		var tmpRespBody respBody
		err = json.Unmarshal(message, &tmpRespBody)
		if err != nil {
			logger.Errorw("Error reading message", err)
			break
		}
		voiceTextStr := tmpRespBody.Result.VoiceTextStr
		fmt.Printf("voiceTextStr: %s\n", voiceTextStr)

		if tmpRespBody.Result.SliceType == 2 {
			h.ch <- voiceTextStr
		}
	}
}

func (m *ASRAPIHandler) Close() error {
	return m.conn.Close()
}
