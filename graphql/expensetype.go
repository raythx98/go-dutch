package graphql

const (
	ExpenseTypeGeneric int16 = iota
	ExpenseTypeRepayment
	ExpenseTypeConversion
)

func expenseTypeString(expenseType int16) string {
	switch expenseType {
	case ExpenseTypeRepayment:
		return "Repayment"
	case ExpenseTypeConversion:
		return "Conversion"
	default:
		return "Generic"
	}
}

func expenseTypeFromString(expenseTypeStr string) int16 {
	switch expenseTypeStr {
	case "Repayment":
		return ExpenseTypeRepayment
	case "Conversion":
		return ExpenseTypeConversion
	default:
		return ExpenseTypeGeneric
	}
}
