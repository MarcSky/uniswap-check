package app

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"time"

	egs "uniswap-bot/internal/ethgasstation"
	"uniswap-bot/internal/model"
	"uniswap-bot/internal/uniswap"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/machinebox/graphql"
)

const (
	uniswapURL = "https://api.thegraph.com/subgraphs/name/uniswap/uniswap-v2"

	uniswapContract = "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D" //mainnet
)

var contractAddress = map[string]string{
	"WETH": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
	"USDC": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
}

type app struct {
	privKey *ecdsa.PrivateKey

	client    *graphql.Client
	ethClient *ethclient.Client
	uni       *uniswap.Uniswap
	egsSvc    egs.EthGasStationSvc
	threshold float64
	contract  common.Address

	tgBotToken  string
	tgChannelID int64
}

func NewUniswap(rpcURL, privKey, tgBotToken string, tgChannelID int64, threshold float64) (*app, error) {
	//connecting
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, err
	}

	contract := common.HexToAddress(uniswapContract)
	uni, err := uniswap.NewUniswap(contract, client)
	if err != nil {
		return nil, err
	}

	ecdsaPrivateKey, err := crypto.HexToECDSA(privKey)
	if err != nil {
		return nil, err
	}

	svc := egs.NewEthGasStationAPI()
	_, _ = svc.GetGasPrices(context.Background())

	return &app{
		client:      graphql.NewClient(uniswapURL),
		ethClient:   client,
		uni:         uni,
		egsSvc:      svc,
		threshold:   threshold,
		privKey:     ecdsaPrivateKey,
		tgBotToken:  tgBotToken,
		tgChannelID: tgChannelID,
		contract:    contract,
	}, nil
}

func (a *app) GetPricePair(pairId string) (*model.Pair, error) {
	var (
		ctx      = context.Background()
		req      = graphql.NewRequest(buildPriceQuery(pairId))
		response = &model.Response{}
	)

	if err := a.client.Run(ctx, req, response); err != nil {
		return nil, fmt.Errorf("failed to load price %s", err)
	}

	price0String := response.Pair.Token0Price
	price0, _ := strconv.ParseFloat(price0String, 64)

	price1String := response.Pair.Token1Price
	price1, _ := strconv.ParseFloat(price1String, 64)

	return &model.Pair{
		Price0: price0,
		Price1: price1,
	}, nil
}

func (a *app) RemoveLiquidity() (string, error) {
	auth, err := bind.NewKeyedTransactorWithChainID(a.privKey, new(big.Int).SetInt64(1))
	if err != nil {
		return "", err
	}

	publicKey := a.privKey.Public()
	publicKeyECDSA, _ := publicKey.(*ecdsa.PublicKey)
	own := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := a.ethClient.PendingNonceAt(context.Background(), own)
	if err != nil {
		return "", err
	}

	data, err := a.uni.RemoveLiquidityETH(
		auth,
		common.HexToAddress(contractAddress["USDC"]),
		new(big.Int).SetInt64(0),
		new(big.Int).SetInt64(0),
		new(big.Int).SetInt64(0),
		own,
		new(big.Int).SetInt64(3600),
	)
	if err != nil {
		return "", err
	}

	e, err := a.egsSvc.GetGasPrices(context.Background())
	if err != nil {
		return "", err
	}

	tx := types.NewTransaction(
		nonce,
		own,
		new(big.Int).SetInt64(0),
		200000,
		new(big.Int).SetInt64(e.Fastest),
		data.Data(),
	)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(new(big.Int).SetInt64(1)), a.privKey)
	if err != nil {
		return "", err
	}

	err = a.ethClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", err
	}

	return data.Hash().String(), nil
}

func (a *app) Swap() (string, error) {
	auth, err := bind.NewKeyedTransactorWithChainID(a.privKey, new(big.Int).SetInt64(1))
	if err != nil {
		return "", err
	}

	publicKey := a.privKey.Public()
	publicKeyECDSA, _ := publicKey.(*ecdsa.PublicKey)
	own := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := a.ethClient.PendingNonceAt(context.Background(), own)
	if err != nil {
		return "", err
	}

	balance, err := a.ethClient.BalanceAt(context.Background(), own, nil)
	if err != nil {
		return "", err
	}

	data, err := a.uni.SwapExactETHForTokens(
		auth,
		new(big.Int).SetInt64(0),
		[]common.Address{
			common.HexToAddress(contractAddress["WETH"]),
			common.HexToAddress(contractAddress["USDC"]),
		},
		own,
		new(big.Int).SetInt64(3600),
	)
	if err != nil {
		return "", err
	}

	e, err := a.egsSvc.GetGasPrices(context.Background())
	if err != nil {
		return "", err
	}

	tx := types.NewTransaction(
		nonce,
		own,
		balance,
		200000,
		new(big.Int).SetInt64(e.Fastest),
		data.Data(),
	)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(new(big.Int).SetInt64(1)), a.privKey)
	if err != nil {
		return "", err
	}

	err = a.ethClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", err
	}

	return data.Hash().String(), nil
}

func (a *app) Run(pairID string) {
	var (
		err    error
		txHash string
		pair   *model.Pair
	)
	for {
		pair, err = a.GetPricePair(pairID)
		if err != nil {
			log.Println("err", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if pair.Price0 < a.threshold {
			continue
		}

		txHash, err = a.RemoveLiquidity()
		if err != nil {
			log.Println("remove liquidity", err)
			_ = a.sendMessage(fmt.Sprintf("remove liquidity %s", err.Error()))
			continue
		}
		log.Println("Liquidity successful removed with hash", txHash)

		txHash, err = a.Swap()
		if err != nil {
			log.Println("swap", err)
			_ = a.sendMessage(fmt.Sprintf("swap %s", err.Error()))
			continue
		}
		log.Println("Swap successful completed with hash", txHash)

		time.Sleep(100 * time.Millisecond)
	}
}

func buildPriceQuery(pairId string) string {
	return fmt.Sprintf(`{ 
		pair(id:"%s") {
			token0Price
			totalSupply
			token0 {
				id
				symbol
				name
				decimals
			}
		} 
	}`, pairId)
}

type sendMessageReqBody struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

func (a *app) sendMessage(msg string) error {
	reqBody := &sendMessageReqBody{
		ChatID:    a.tgChannelID,
		Text:      msg,
		ParseMode: "HTML",
	}

	sendBody, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	res, err := http.Post(fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", a.tgBotToken), "application/json", bytes.NewBuffer(sendBody))
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return errors.New("unexpected problem")
	}

	return nil
}
