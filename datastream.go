package main

import lksdk "github.com/livekit/server-sdk-go/v2"

func datastream(room *lksdk.Room, dataStreamChan chan string) {
	for {
		select {
		case text, ok := <-dataStreamChan:
			if !ok {
				return
			}
			topic := "text-read-iter"
			room.LocalParticipant.SendText(text, lksdk.StreamTextOptions{
				Topic: topic,
			})
		}
	}
}
