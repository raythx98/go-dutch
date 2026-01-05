package pghelper

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

func Int8(userId *int64) pgtype.Int8 {
	if userId == nil {
		return pgtype.Int8{Valid: false}
	}

	return pgtype.Int8{Int64: *userId, Valid: true}
}

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
