package graphql

import (
	"github.com/raythx98/go-dutch/sqlc/db"
	"github.com/raythx98/go-dutch/tools/resources"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	resources.Tools
	DbQuery *db.Queries
}

func NewResolver(tools resources.Tools) *Resolver {
	return &Resolver{
		Tools:   tools,
		DbQuery: db.New(tools.Db.Pool()),
	}
}
