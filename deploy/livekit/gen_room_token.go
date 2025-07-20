package main

import (
	"fmt"
	"time"
	//
	"github.com/livekit/protocol/auth"
)

func NewAccessToken(roomName, pID string) (string, error) {

	// 这东西泄露也没啥事，每次部署生成都不一样
	apiKey := "API4Lpvvg2Hfynv"                                 // 填写你刚才代码生成的apiKey
	apiSecret := "BekllZvfAd7SsiGVNU0OQT45Ahe2tod8TwV90By5WlyB" // 填写你刚才代码生成的apiSecret

	at := auth.NewAccessToken(apiKey, apiSecret)
	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     roomName,
	}
	at.SetVideoGrant(grant).
		SetIdentity(pID).
		SetName(pID).
		SetValidFor(time.Hour * 24 * 30) // 30天过期

	return at.ToJWT()
}

func main() {
	token, _ := NewAccessToken("room01", "participant-client") // 房间名，用户参会者名字
	fmt.Printf(token)
}
