package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"github.com/raythx98/go-dutch/graphql"
	"github.com/raythx98/go-dutch/tools/config"
	"github.com/raythx98/go-dutch/tools/resources"
	"github.com/raythx98/gohelpme/errorhelper"
	"github.com/raythx98/gohelpme/middleware"
	"github.com/raythx98/gohelpme/tool/logger"
	"github.com/raythx98/gohelpme/tool/reqctx"
	"github.com/vektah/gqlparser/v2/parser"

	gql "github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"gopkg.in/yaml.v3"
)

func main() {
	ctx := context.Background()

	cfg := config.Load()
	fmt.Printf("configs loaded: %+v\n", cfg)

	tools := resources.CreateTools(cfg, ctx)
	defer tools.Db.Pool().Close()

	srv := handler.New(graphql.NewExecutableSchema(graphql.Config{
		Resolvers: graphql.NewResolver(tools),
		Directives: graphql.DirectiveRoot{
			Auth: func(ctx context.Context, obj any, next gql.Resolver) (any, error) {
				userId := reqctx.GetValue(ctx).UserId
				if userId == nil || *userId == 0 {
					return nil, errorhelper.NewAuthError(nil)
				}

				// or let it pass through
				return next(ctx)
			},
		},
	}))
	srv.SetRecoverFunc(func(ctx context.Context, err any) (userMessage error) {
		panicErr := fmt.Errorf("[panic] %v", err)
		reqctx.GetValue(ctx).
			SetError(panicErr).
			SetErrorStack(debug.Stack())
		return panicErr
	})

	srv.SetErrorPresenter(func(ctx context.Context, e error) *gqlerror.Error {
		err := gql.DefaultErrorPresenter(ctx, e)
		if err.Extensions == nil {
			err.Extensions = make(map[string]interface{})
		}

		var myErr *errorhelper.AppError
		var authErr *errorhelper.AuthError
		if errors.As(e, &myErr) {
			err.Message = myErr.Message
			err.Extensions["code"] = myErr.Code
		} else if errors.As(e, &authErr) {
			err.Message = "Unauthorized"
			err.Extensions["code"] = 401
		} else {
			tools.Log.Error(ctx, "internal server error", logger.WithError(e))
			err.Message = "Something went wrong, please try again later"
			err.Extensions["code"] = 500
		}

		return err
	})

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	mux := http.NewServeMux()

	// Load Rate Limit Config
	var rlConfig middleware.Config
	rlFile, err := os.ReadFile("ratelimit.yaml")
	if err != nil {
		tools.Log.Warn(ctx, "failed to load ratelimit.yaml, using defaults", logger.WithError(err))
		rlConfig = middleware.Config{
			Default: middleware.RateConfig{Rate: 1, Burst: 2},
		}
	} else {
		if err := yaml.Unmarshal(rlFile, &rlConfig); err != nil {
			log.Fatalf("failed to parse ratelimit.yaml: %v", err)
		}
	}

	rateLimiter := middleware.NewRateLimiter(rlConfig, tools.Log, func(r *http.Request) (string, string) {
		identifier := middleware.ExtractIP(r)
		if reqCtx := reqctx.GetValue(r.Context()); reqCtx != nil && reqCtx.UserId != nil {
			identifier = fmt.Sprintf("user:%d", *reqCtx.UserId)
		}

		var gqlQueryString string
		var clientOperationName string

		if r.Method == http.MethodPost {
			bodyBytes, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

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
			if err == nil {
				var op *ast.OperationDefinition
				if clientOperationName != "" {
					op = parsedQuery.Operations.ForName(clientOperationName)
				} else if len(parsedQuery.Operations) > 0 {
					for _, o := range parsedQuery.Operations {
						op = o
						break
					}
				}

				if op != nil && len(op.SelectionSet) > 0 {
					if field, ok := op.SelectionSet[0].(*ast.Field); ok {
						opType := strings.Title(string(op.Operation))
						derivedOperationName = fmt.Sprintf("%s.%s", opType, field.Name)
					}
				}
			}
		}

		if derivedOperationName == "" {
			derivedOperationName = "anonymous_or_unparsable"
		}

		return identifier, derivedOperationName
	})

	queryHandler := middleware.Chain(srv.ServeHTTP, []func(http.HandlerFunc) http.HandlerFunc{
		middleware.CORS,
		rateLimiter.RateLimit,
		middleware.AddRequestId,
		middleware.ReqCtx,
		middleware.JwtSubject(tools.Jwt),
		middleware.Log(tools.Log, middleware.LogConfig{
			RedactedPaths: []string{
				"request.body.variables.password",
				"request.headers.Authorization",
				"response.body.data.register.token",
				"response.body.data.login.token",
			},
		}),
	}...)

	mux.Handle("/", playground.Handler("GraphQL playground", "/query"))
	mux.Handle("/query", queryHandler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	tools.Log.Info(ctx, "starting server",
		logger.WithField("host", "localhost"),
		logger.WithField("port", cfg.ServerPort))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.ServerPort), mux))
}
