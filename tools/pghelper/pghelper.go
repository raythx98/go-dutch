package pghelper

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

func Time(time *time.Time) pgtype.Timestamp {
	if time == nil {
		return pgtype.Timestamp{Valid: false}
	}

	return pgtype.Timestamp{Time: *time, Valid: true}
}

func FromDecimal(d decimal.Decimal) pgtype.Numeric {
	return pgtype.Numeric{
		Int:   d.Coefficient(),
		Exp:   d.Exponent(),
		Valid: true,
	}
}

func Decimal(n pgtype.Numeric) decimal.Decimal {
	return decimal.NewFromBigInt(n.Int, n.Exp)
}
