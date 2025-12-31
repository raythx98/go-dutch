package resources

import (
	"context"
	"time"

	"github.com/raythx98/go-dutch/tools/config"
	cryptoTool "github.com/raythx98/go-dutch/tools/crypto"
	postgresTool "github.com/raythx98/go-dutch/tools/postgres"
	"github.com/raythx98/go-dutch/tools/zerologger"

	"github.com/raythx98/gohelpme/tool/crypto"
	"github.com/raythx98/gohelpme/tool/jwthelper"
	"github.com/raythx98/gohelpme/tool/logger"
	"github.com/raythx98/gohelpme/tool/postgres"
)

type Tools struct {
	Log    logger.ILogger
	Db     postgresTool.IPostgres
	Jwt    jwthelper.IJwt
	Crypto cryptoTool.ICrypto
}

func CreateTools(cfg *config.Specification, ctx context.Context) Tools {
	log := zerologger.New(cfg.Debug)
	return Tools{
		Log: log,
		Db:  postgres.New(ctx, cfg, log),
		Jwt: jwthelper.New(jwthelper.Config{
			Issuer:               "raythx98@gmail.com",
			Audiences:            []string{"raythx98@gmail.com"},
			AccessTokenValidity:  24 * time.Hour,
			RefreshTokenValidity: 0,
		}, cfg),
		Crypto: crypto.New(crypto.DefaultConfig()),
	}
}
