package wallet

import (
	"encoding/json"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/proxy"
)

type HttpClient interface {
	Get(string) (*http.Response, error)
}

type feeCache struct {
	fees        *Fees
	lastUpdated time.Time
}

type Fees struct {
	Priority uint64 `json:"priority"`
	Normal   uint64 `json:"normal"`
	Economic uint64 `json:"economic"`
}

type FeeProvider struct {
	MaxFee      uint64
	PriorityFee uint64
	NormalFee   uint64
	EconomicFee uint64
	FeeAPI      string

	HttpClient HttpClient

	cache *feeCache
}

func NewFeeProvider(maxFee, priorityFee, normalFee, economicFee uint64, feeAPI string, proxy proxy.Dialer) *FeeProvider {
	fp := FeeProvider{
		MaxFee:      maxFee,
		PriorityFee: priorityFee,
		NormalFee:   normalFee,
		EconomicFee: economicFee,
		FeeAPI:      feeAPI,
		cache:       new(feeCache),
	}
	dial := net.Dial
	if proxy != nil {
		dial = proxy.Dial
	}
	tbTransport := &http.Transport{Dial: dial}
	httpClient := &http.Client{Transport: tbTransport, Timeout: time.Second * 10}
	fp.HttpClient = httpClient
	return &fp
}

func (fp *FeeProvider) GetFeePerByte(feeLevel FeeLevel) uint64 {
	if fp.FeeAPI == "" {
		return fp.defaultFee(feeLevel)
	}
	fees := new(Fees)
	if time.Since(fp.cache.lastUpdated) > time.Minute {
		resp, err := fp.HttpClient.Get(fp.FeeAPI)
		if err != nil {
			return fp.defaultFee(feeLevel)
		}

		defer resp.Body.Close()

		err = json.NewDecoder(resp.Body).Decode(&fees)
		if err != nil {
			return fp.defaultFee(feeLevel)
		}
		fp.cache.lastUpdated = time.Now()
		fp.cache.fees = fees
	} else {
		fees = fp.cache.fees
	}
	switch feeLevel {
	case PRIOIRTY:
		return fp.selectFee(fees.Priority, PRIOIRTY)
	case NORMAL:
		return fp.selectFee(fees.Normal, PRIOIRTY)
	case ECONOMIC:
		return fp.selectFee(fees.Economic, PRIOIRTY)
	case FEE_BUMP:
		return fp.selectFee(fees.Priority, PRIOIRTY)
	default:
		return fp.NormalFee
	}
}

func (fp *FeeProvider) selectFee(fee uint64, feeLevel FeeLevel) uint64 {
	if fee > fp.MaxFee {
		return fp.MaxFee
	} else if fee == 0 {
		return fp.defaultFee(feeLevel)
	} else {
		return fee
	}
}

func (fp *FeeProvider) defaultFee(feeLevel FeeLevel) uint64 {
	switch feeLevel {
	case PRIOIRTY:
		return fp.PriorityFee
	case NORMAL:
		return fp.NormalFee
	case ECONOMIC:
		return fp.EconomicFee
	case FEE_BUMP:
		return fp.PriorityFee
	default:
		return fp.NormalFee
	}
}
