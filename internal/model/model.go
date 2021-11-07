package model

type Pair struct {
	Price0 float64
	Price1 float64
}

type Token struct {
	ID         string  `json:"id"`
	Symbol     string  `json:"symbol"`
	Name       string  `json:"name"`
	Price      string  `json:"price"`
	EthPrice   float64 `json:"ethPrice"`
	PriceFloat float64 `json:"priceFloat"`
	MarketCap  string  `json:"marketCap"`
	Decimals   string  `json:"decimals"`
}

type Response struct {
	Pair struct {
		Token0Price string `json:"token0Price"`
		Token1Price string `json:"token1Price"`
		TotalSupply string `json:"totalSupply"`
		Token       Token  `json:"token0"`
	} `json:"pair"`
}
