package graphql

const (
	ExpenseTypeGeneric int16 = iota
	ExpenseTypeRepayment
)

func expenseTypeString(expenseType int16) string {
	switch expenseType {
	case ExpenseTypeRepayment:
		return "Repayment"
	default:
		return "Generic"
	}
}

func expenseTypeFromString(expenseTypeStr string) int16 {
	switch expenseTypeStr {
	case "Repayment":
		return ExpenseTypeRepayment
	default:
		return ExpenseTypeGeneric
	}
}
