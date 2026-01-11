package ratelimit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/raythx98/gohelpme/tool/logger"
	"golang.org/x/time/rate"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

var opNameRegex = regexp.MustCompile(`(?i)^\s*(query|mutation|subscription)\s+(\w+)`)

type Config struct {
	Default RateConfig            `yaml:"default"`
	Ops     map[string]RateConfig `yaml:"operations"`
}

type RateConfig struct {
	Rate  float64 `yaml:"rate"`
	Burst int     `yaml:"burst"`
}

type RateLimiter struct {
	config  Config
	log     logger.ILogger
	ips     sync.Map // map[string]*rate.Limiter (key: "ip:operation")
	cleanup *time.Ticker
}

func NewRateLimiter(cfg Config, log logger.ILogger) *RateLimiter {
	rl := &RateLimiter{
		config:  cfg,
		log:     log,
		cleanup: time.NewTicker(1 * time.Minute),
	}
	go rl.startCleanup()
	return rl
}

func (rl *RateLimiter) startCleanup() {
	for range rl.cleanup.C {
		// In a real implementation we would track last seen time.
		// For simplicity/MVP, we are just clearing the map to prevent leaks.
		// A better approach is to wrap the limiter in a struct with LastSeen.
		rl.ips.Range(func(key, value any) bool {
			rl.ips.Delete(key)
			return true
		})
	}
}

func (rl *RateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract IP
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.Header.Get("X-Real-IP")
		}
		if ip == "" {
			ip, _, _ = net.SplitHostPort(r.RemoteAddr)
			if ip == "" {
				ip = r.RemoteAddr
			}
		} else {
			// X-Forwarded-For can be "client, proxy1, proxy2"
			if strings.Contains(ip, ",") {
				ip = strings.TrimSpace(strings.Split(ip, ",")[0])
			}
		}

		// Extract Operation Name from GraphQL Query
		var gqlQueryString string      // This will hold the raw GraphQL query string
		var clientOperationName string // Client provided operationName, if any

		if r.Method == http.MethodPost {
			bodyBytes, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Restore body

			var gqlReq struct {
				OperationName string `json:"operationName"`
				Query         string `json:"query"`
			}
			if err := json.Unmarshal(bodyBytes, &gqlReq); err == nil {
				clientOperationName = gqlReq.OperationName
				gqlQueryString = gqlReq.Query
			}
		} else if r.Method == http.MethodGet {
			clientOperationName = r.URL.Query().Get("operationName")
			gqlQueryString = r.URL.Query().Get("query")
		}

		derivedOperationName := ""
		if gqlQueryString != "" {
			parsedQuery, err := parser.ParseQuery(&ast.Source{Input: gqlQueryString})
			if err != nil {
				rl.log.Error(ctx, "failed to parse GraphQL query for rate limiting", logger.WithError(err), logger.WithField("query", gqlQueryString))
			} else {
				var op *ast.OperationDefinition
				if clientOperationName != "" {
					op = parsedQuery.Operations.ForName(clientOperationName)
				} else if len(parsedQuery.Operations) > 0 {
					// For anonymous queries or if clientOperationName is missing, pick the first operation
					// Iterate over map to get first element (order is not guaranteed, but for single op, it's fine)
					for _, o := range parsedQuery.Operations {
						op = o
						break
					}
				}

				if op != nil && len(op.SelectionSet) > 0 {
					// Get the first root field name
					if field, ok := op.SelectionSet[0].(*ast.Field); ok {
						// Format: "Query.fieldName" or "Mutation.fieldName"
						derivedOperationName = fmt.Sprintf("%s.%s", strings.Title(string(op.Operation)), field.Name)
					}
				}
			}
		}

		key := ip
		if derivedOperationName != "" {
			key += ":" + derivedOperationName
		} else {
			// Fallback if no specific operation could be derived.
			// This covers anonymous queries without root fields (unlikely), or parsing errors.
			// This will use the default rate limit.
			key += ":" + "anonymous_or_unparsable"
		}
		limitConfig := rl.config.Default
		if opConfig, ok := rl.config.Ops[derivedOperationName]; ok {
			limitConfig = opConfig
		}

		limiter, _ := rl.ips.LoadOrStore(key, rate.NewLimiter(rate.Limit(limitConfig.Rate), limitConfig.Burst))

		if !limiter.(*rate.Limiter).Allow() {
			rl.log.Warn(ctx, "rate limit exceeded",
				logger.WithField("ip", ip),
				logger.WithField("operation", derivedOperationName))
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Rate limit exceeded"))
			return
		}

		next(w, r)
	}
}
