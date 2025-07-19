package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func llm(asrToLLMChan chan string, dataStreamChan chan string) {
	for {
		select {
		case text, ok := <-asrToLLMChan:
			if !ok {
				return
			}
			println("llm: ", text)
			requestLLM(text, dataStreamChan)
		}

	}
}

func requestLLM(query string, dataStreamChan chan string) {
	// 定义请求体（使用 map 构建）
	body := map[string]interface{}{
		"model": "ernie-3.5-8k",
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": "你是一个智能问答助手，回答我的问题。要求言简意赅，每次回答不能超过100个字。",
			},
			{
				"role":    "user",
				"content": query,
			},
		},
		"stream": true,
	}

	// 转换为 JSON
	jsonBody, _ := json.Marshal(body)

	// 创建请求
	req, err := http.NewRequest("POST", modelUrl, bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", modelToken)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Unexpected status code:", resp.StatusCode)
		return
	}

	// 使用 bufio 逐行读取响应体
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading line:", err)
			break
		}

		if strings.Contains(line, "[DONE]") {
			break
		}

		// 去除前缀 "data: "，并去除空格
		if strings.HasPrefix(line, "data: ") {
			jsonStr := strings.TrimSpace(line[6:]) // 去掉 "data: " 前缀

			if jsonStr == "" {
				continue
			}

			// 解析 JSON 数据
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(jsonStr), &event); err != nil {
				fmt.Println("Error unmarshalling JSON:", err)
				continue
			}

			// 提取 content 字段
			if choices, ok := event["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if delta, ok := choice["delta"].(map[string]interface{}); ok {
						if content, ok := delta["content"].(string); ok {
							fmt.Println("Extracted content:", content)
							dataStreamChan <- content
						}
					}
				}
			}
		}
	}
}
