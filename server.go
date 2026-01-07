package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/raythx98/go-dutch/graphql"
	"github.com/raythx98/go-dutch/tools/config"
	"github.com/raythx98/go-dutch/tools/resources"

	"github.com/raythx98/gohelpme/errorhelper"
	"github.com/raythx98/gohelpme/middleware"
	"github.com/raythx98/gohelpme/tool/logger"
	"github.com/raythx98/gohelpme/tool/reqctx"

	gql "github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
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

	queryHandler := middleware.Chain(srv.ServeHTTP, []func(http.HandlerFunc) http.HandlerFunc{
		middleware.CORS,
		middleware.AddRequestId,
		middleware.ReqCtx,
		middleware.JwtSubject(tools.Jwt),
		middleware.Log(tools.Log),
	}...)

	mux.Handle("/", playground.Handler("GraphQL playground", "/query"))
	mux.Handle("/query", queryHandler)

	tools.Log.Info(ctx, "starting server",
		logger.WithField("host", "localhost"),
		logger.WithField("port", cfg.ServerPort))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.ServerPort), mux))
}
