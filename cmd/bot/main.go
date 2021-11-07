package main

import (
	"os"
	"strconv"

	"uniswap-bot/internal/app"
)

const (
	pairID = "0xb4e16d0168e52d35cacd2c6185b44281ec28c9dc" // ETH/USDC
)

func main() {
	var (
		rpcURL       = os.Getenv("ETH_RPC_URL")
		privateKey   = os.Getenv("PRIVATE_KEY")
		tgBotToken   = os.Getenv("TG_BOT_TOKEN")
		tgChannelID  = os.Getenv("TG_CHANNEL_ID")
		ethThreshold = os.Getenv("ETH_THRESHOLD")
	)

	channelID, _ := strconv.Atoi(tgChannelID)
	threshold, _ := strconv.ParseFloat(ethThreshold, 64)
	u, err := app.NewUniswap(rpcURL, privateKey, tgBotToken, int64(channelID), threshold)
	if err != nil {
		panic(err)
	}

	u.Run(pairID)
}
