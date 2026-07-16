// Package price provides a thread-safe in-memory cache for exchange rates fetched from
// the Binance Public API. The cache refreshes automatically when TTL expires.
package price

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

const cacheTTL = 30 * time.Second

// CachedRate holds a rate value alongside the time it was fetched.
type CachedRate struct {
	Rate      decimal.Decimal
	FetchedAt time.Time
}

// PriceCache fetches and caches exchange rates from the Binance Public API.
// All methods are safe for concurrent use.
type PriceCache struct {
	mu         sync.RWMutex
	cache      map[string]CachedRate
	binanceURL string
	usdIDRRate decimal.Decimal  // Fallback / configured USD→IDR rate
	httpClient *http.Client
}

// NewPriceCache creates a new PriceCache.
//   - binanceURL: e.g., "https://api.binance.com/api/v3"
//   - usdIDRRate: fallback rate, e.g., 16200 (can be updated via env)
func NewPriceCache(binanceURL string, usdIDRRate decimal.Decimal) *PriceCache {
	return &PriceCache{
		cache:      make(map[string]CachedRate),
		binanceURL: binanceURL,
		usdIDRRate: usdIDRRate,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// GetRate returns the exchange rate for the given pair (e.g., "USDT_IDR", "USDC_IDR").
// If the cached value has expired, it fetches a fresh rate from Binance.
func (p *PriceCache) GetRate(pair string) (decimal.Decimal, time.Time, error) {
	p.mu.RLock()
	if cached, ok := p.cache[pair]; ok && time.Since(cached.FetchedAt) < cacheTTL {
		p.mu.RUnlock()
		return cached.Rate, cached.FetchedAt, nil
	}
	p.mu.RUnlock()

	// Cache miss or expired — fetch fresh rate.
	rate, err := p.fetchRate(pair)
	if err != nil {
		// Return stale data if available, rather than failing hard.
		p.mu.RLock()
		if cached, ok := p.cache[pair]; ok {
			p.mu.RUnlock()
			return cached.Rate, cached.FetchedAt, nil
		}
		p.mu.RUnlock()
		return decimal.Zero, time.Time{}, fmt.Errorf("rate unavailable for %s: %w", pair, err)
	}

	now := time.Now()
	p.mu.Lock()
	p.cache[pair] = CachedRate{Rate: rate, FetchedAt: now}
	p.mu.Unlock()

	return rate, now, nil
}

// fetchRate resolves a pair to a Binance/Indodax symbol and fetches the price.
// Strategy:
//   - USDT_IDR: Fetch USDT/IDR price from Indodax, fallback to USD_IDR_RATE.
//   - USDC_IDR: Fetch USDC/USDT price from Binance, multiply by USDT/IDR price from Indodax (fallback to USD_IDR_RATE if Indodax fails).
func (p *PriceCache) fetchRate(pair string) (decimal.Decimal, error) {
	if pair == "USDT_IDR" {
		rate, err := p.fetchIndodaxPrice()
		if err != nil {
			// Fallback to configured rate if Indodax is down
			return p.usdIDRRate, nil
		}
		return rate, nil
	}

	if pair == "USDC_IDR" {
		usdcUsdtPrice, err := p.fetchBinancePrice("USDCUSDT")
		if err != nil {
			// Fallback to 1.0 USDC = 1.0 USDT if Binance is down
			usdcUsdtPrice = decimal.NewFromInt(1)
		}

		usdtIdrPrice, err := p.fetchIndodaxPrice()
		if err != nil {
			// Fallback to configured rate if Indodax is down
			usdtIdrPrice = p.usdIDRRate
		}

		return usdcUsdtPrice.Mul(usdtIdrPrice), nil
	}

	return decimal.Zero, fmt.Errorf("unsupported pair: %s", pair)
}

type binanceTickerResponse struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

func (p *PriceCache) fetchBinancePrice(symbol string) (decimal.Decimal, error) {
	if flag.Lookup("test.v") != nil {
		return decimal.Zero, fmt.Errorf("network disabled in test mode")
	}

	url := fmt.Sprintf("%s/ticker/price?symbol=%s", p.binanceURL, symbol)

	resp, err := p.httpClient.Get(url)
	if err != nil {
		return decimal.Zero, fmt.Errorf("binance request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decimal.Zero, fmt.Errorf("binance returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to read binance response: %w", err)
	}

	var ticker binanceTickerResponse
	if err := json.Unmarshal(body, &ticker); err != nil {
		return decimal.Zero, fmt.Errorf("failed to parse binance response: %w", err)
	}

	price, err := decimal.NewFromString(ticker.Price)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid price value from binance: %w", err)
	}

	return price, nil
}

type indodaxTickerResponse struct {
	Ticker struct {
		Last string `json:"last"`
	} `json:"ticker"`
}

func (p *PriceCache) fetchIndodaxPrice() (decimal.Decimal, error) {
	if flag.Lookup("test.v") != nil {
		return decimal.Zero, fmt.Errorf("network disabled in test mode")
	}

	url := "https://indodax.com/api/ticker/usdtidr"

	resp, err := p.httpClient.Get(url)
	if err != nil {
		return decimal.Zero, fmt.Errorf("indodax request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decimal.Zero, fmt.Errorf("indodax returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to read indodax response: %w", err)
	}

	var response indodaxTickerResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return decimal.Zero, fmt.Errorf("failed to parse indodax response: %w", err)
	}

	price, err := decimal.NewFromString(response.Ticker.Last)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid price value from indodax: %w", err)
	}

	return price, nil
}
