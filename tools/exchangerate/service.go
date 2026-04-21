package exchangerate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raythx98/go-dutch/sqlc/db"
	"github.com/raythx98/gohelpme/tool/logger"
)

// exchangeRateLockKey is the PostgreSQL advisory lock key for serialising API fetches.
// Must not collide with any other advisory lock in the codebase.
const exchangeRateLockKey int64 = 8200001

const snapshotTTL = 24 * time.Hour
const apiTimeout = 10 * time.Second

// Result holds a parsed exchange rate snapshot.
type Result struct {
	FetchedAt             time.Time
	Rates                 map[string]float64
	UnsupportedCurrencies []string
}

// Service fetches and caches USD-base exchange rates.
type Service struct {
	pool        *pgxpool.Pool
	pooledQuery *db.Queries
	apiKey      string
	log         logger.ILogger
}

// New creates a Service.
func New(pool *pgxpool.Pool, apiKey string, log logger.ILogger) *Service {
	return &Service{
		pool:        pool,
		pooledQuery: db.New(pool),
		apiKey:      apiKey,
		log:         log,
	}
}

// GetOrRefresh returns a fresh or cached exchange rate snapshot.
// Uses a PostgreSQL advisory lock to prevent concurrent API calls across instances.
func (s *Service) GetOrRefresh(ctx context.Context) (*Result, error) {
	// Fast path: check snapshot without lock.
	snapshot, err := s.pooledQuery.GetExchangeRateSnapshot(ctx, "USD")
	if err == nil && time.Since(snapshot.FetchedAt.Time) < snapshotTTL {
		return s.parseSnapshot(ctx, snapshot)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Slow path: pin a connection and acquire advisory lock.
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	qtx := db.New(conn)

	if err := qtx.AcquireExchangeRateLock(ctx, exchangeRateLockKey); err != nil {
		return nil, err
	}
	defer func() {
		_ = qtx.ReleaseExchangeRateLock(context.Background(), exchangeRateLockKey)
	}()

	// Double-check: another instance may have refreshed while we waited.
	snapshot, err = qtx.GetExchangeRateSnapshot(ctx, "USD")
	if err == nil && time.Since(snapshot.FetchedAt.Time) < snapshotTTL {
		return s.parseSnapshot(ctx, snapshot)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Fetch from exchangerate-api.com.
	freshRates, fetchErr := s.fetchFromAPI(ctx)
	if fetchErr != nil {
		s.log.Warn(ctx, fmt.Sprintf("exchange rate API fetch failed: %v", fetchErr))
		// Fall back to stale snapshot if available.
		if err == nil {
			return s.parseSnapshot(ctx, snapshot)
		}
		return nil, fmt.Errorf("exchange rate unavailable: %w", fetchErr)
	}

	ratesBytes, err := json.Marshal(freshRates)
	if err != nil {
		return nil, err
	}

	snapshot, err = qtx.UpsertExchangeRateSnapshot(ctx, db.UpsertExchangeRateSnapshotParams{
		BaseCurrencyCode: "USD",
		Rates:            ratesBytes,
	})
	if err != nil {
		return nil, err
	}

	result := &Result{
		FetchedAt: snapshot.FetchedAt.Time,
		Rates:     freshRates,
	}
	result.UnsupportedCurrencies = s.findUnsupported(ctx, freshRates)
	return result, nil
}

type apiResponse struct {
	Result            string             `json:"result"`
	ConversionRates   map[string]float64 `json:"conversion_rates"`
}

func (s *Service) fetchFromAPI(ctx context.Context) (map[string]float64, error) {
	if s.apiKey == "" {
		return nil, errors.New("exchange rate API key not configured")
	}

	reqCtx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	url := fmt.Sprintf("https://v6.exchangerate-api.com/v6/%s/latest/USD", s.apiKey)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}
	if apiResp.Result != "success" {
		return nil, fmt.Errorf("API result: %s", apiResp.Result)
	}

	return apiResp.ConversionRates, nil
}

func (s *Service) parseSnapshot(ctx context.Context, snapshot db.ExchangeRateSnapshot) (*Result, error) {
	var rates map[string]float64
	if err := json.Unmarshal(snapshot.Rates, &rates); err != nil {
		return nil, err
	}
	result := &Result{
		FetchedAt: snapshot.FetchedAt.Time,
		Rates:     rates,
	}
	result.UnsupportedCurrencies = s.findUnsupported(ctx, rates)
	return result, nil
}

func (s *Service) findUnsupported(ctx context.Context, rates map[string]float64) []string {
	currencies, err := s.pooledQuery.GetCurrencies(ctx)
	if err != nil {
		return nil
	}
	var unsupported []string
	for _, c := range currencies {
		if _, ok := rates[c.Code]; !ok {
			s.log.Warn(ctx, fmt.Sprintf("exchange rate unavailable for currency: %s", c.Code))
			unsupported = append(unsupported, c.Code)
		}
	}
	return unsupported
}
