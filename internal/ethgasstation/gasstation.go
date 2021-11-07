package ethgasstation

import (
	"context"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"time"

	"uniswap-bot/internal/ethgasstation/cache"
	"uniswap-bot/internal/ethgasstation/cache/memory"

	"github.com/mailru/easyjson"
)

const (
	apiEthGasStationURL = "https://ethgasstation.info/api/ethgasAPI.json"
)

type (
	api struct {
		client    *http.Client
		cacheTime time.Duration
		storage   cache.Storage
	}

	EthGasStationSvc interface { // nolint:golint
		GetGasPrices(ctx context.Context) (*Station, error)
	}
)

func (c *api) req(ctx context.Context) (*Station, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiEthGasStationURL, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var gasPrices Station
	if err := easyjson.Unmarshal(body, &gasPrices); err != nil {
		return nil, err
	}

	return &gasPrices, nil
}

func (c *api) GetGasPrices(ctx context.Context) (*Station, error) {
	cachedPrices, ok := c.storage.Get()
	if ok {
		s := c.mappingMemoryStationToStation(cachedPrices)
		return &s, nil
	}

	actualPrices, err := c.req(ctx)
	if err != nil {
		s := c.mappingMemoryStationToStation(c.storage.GetWithoutExpiration())
		return &s, err
	}

	c.storage.Set(c.mappingStationToMemoryStation(actualPrices), c.cacheTime)
	return actualPrices, nil
}

func NewEthGasStationAPI() EthGasStationSvc {
	defaultTransport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        20,
		MaxIdleConnsPerHost: 20,
		TLSHandshakeTimeout: 15 * time.Second,
	}

	client := &http.Client{
		Transport: defaultTransport,
		Timeout:   25 * time.Second,
	}

	return &api{
		client:    client,
		cacheTime: 10 * time.Second,
		storage:   memory.NewStorage(),
	}
}

func (c api) mappingStationToMemoryStation(s *Station) memory.Station {
	return memory.Station{
		Fast:      GWei2Wei(s.Fast / 10),    //nolint:gomnd
		Fastest:   GWei2Wei(s.Fastest / 10), //nolint:gomnd
		SafeLow:   GWei2Wei(s.SafeLow / 10), //nolint:gomnd
		Average:   GWei2Wei(s.Average / 10), //nolint:gomnd
		BlockTime: s.BlockTime,
	}
}

func (c api) mappingMemoryStationToStation(s *memory.Station) Station {
	return Station{
		Fast:      s.Fast,
		Fastest:   s.Fastest,
		SafeLow:   s.SafeLow,
		Average:   s.Average,
		BlockTime: s.BlockTime,
	}
}

func GWei2Wei(v int64) int64 {
	value := big.NewInt(0).SetInt64(v)
	value.Mul(value, big.NewInt(0).SetUint64(1000000000)) //nolint:gomnd
	return value.Int64()
}
