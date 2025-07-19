package main

import (
	"github.com/joho/godotenv"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
}

func connectToLKRoom(cb *lksdk.RoomCallback) (*lksdk.Room, error) {
	room, err := lksdk.ConnectToRoom(host, lksdk.ConnectInfo{
		APIKey:              apiKey,
		APISecret:           apiSecret,
		RoomName:            roomName,
		ParticipantIdentity: participantIdentity,
	}, cb)
	if err != nil {
		return nil, err
	}
	return room, nil
}
