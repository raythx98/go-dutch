package model

import (
	"fmt"
	"io"

	"github.com/99designs/gqlgen/graphql"
	"github.com/shopspring/decimal"
)

func MarshalDecimal(n decimal.Decimal) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		w.Write([]byte(n.String()))
	})
}

func UnmarshalDecimal(v interface{}) (decimal.Decimal, error) {
	str, ok := v.(string)
	if !ok {
		return decimal.Decimal{}, fmt.Errorf("decimal must be a string")
	}

	amount, err := decimal.NewFromString(str)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("invalid decimal string: %v", err)
	}

	return amount, nil
}
