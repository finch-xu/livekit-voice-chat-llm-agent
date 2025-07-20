package main

import (
	"fmt"
	"github.com/livekit/protocol/utils"
)

func main() {
	apiKey := utils.NewGuid(utils.APIKeyPrefix)
	apiSecret := utils.RandomSecret()

	fmt.Printf("apiKey: %s\n", apiKey)
	fmt.Printf("apiSecret: %s\n", apiSecret)
}
