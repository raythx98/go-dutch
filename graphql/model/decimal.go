package model

import (
	"fmt"
	"io"

	"github.com/99designs/gqlgen/graphql"
	"github.com/jackc/pgx/v5/pgtype"
)

func MarshalPgNumeric(n pgtype.Numeric) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		json, err := n.MarshalJSON()
		if err != nil {
			w.Write([]byte("null"))
			return
		}

		w.Write(json)
	})
}

func UnmarshalPgNumeric(v interface{}) (pgtype.Numeric, error) {
	str, ok := v.(string)
	if !ok {
		return pgtype.Numeric{}, fmt.Errorf("decimal must be a string")
	}

	var n pgtype.Numeric
	if err := n.Scan(str); err != nil {
		return pgtype.Numeric{}, err
	}

	return n, nil
}
