package graphql

import (
	"context"
	"slices"

	"github.com/raythx98/go-dutch/graphql/model"
	"github.com/raythx98/go-dutch/sqlc/db"
	"github.com/raythx98/go-dutch/tools/pghelper"
	"github.com/raythx98/gohelpme/errorhelper"
	"github.com/raythx98/gohelpme/tool/reqctx"
	"github.com/shopspring/decimal"
)

func getActionTaker(ctx context.Context) int64 {
	return *reqctx.GetValue(ctx).UserId
}

func checkIsGroupMember(ctx context.Context, dbQuery *db.Queries, groupID int64, userId int64) ([]db.User, error) {
	members, err := dbQuery.GetGroupMembers(ctx, groupID)
	if err != nil {
		return members, err
	}

	if !slices.ContainsFunc(members, func(n db.User) bool { return n.ID == userId }) {
		return members, errorhelper.NewAppError(NotGroupMember, Messages[NotGroupMember], nil)
	}

	return members, nil
}
func getCurrency(ctx context.Context, dbQuery *db.Queries, err error, currencyId int64) (db.Currency, error) {
	currencies, err := dbQuery.GetCurrenciesByIds(ctx, []int64{currencyId})
	if err != nil {
		return db.Currency{}, err
	}
	if len(currencies) == 0 {
		return db.Currency{}, errorhelper.NewAppError(CurrencyNotSupported, Messages[CurrencyNotSupported], nil)
	}
	return currencies[0], nil
}

func fetchUsersMap(ctx context.Context, dbQuery *db.Queries, input model.ExpenseInput) (map[int64]db.User, error) {
	userIds := make([]int64, 0)
	for _, payer := range input.Payers {
		if !slices.Contains(userIds, payer.UserID) {
			userIds = append(userIds, payer.UserID)
		}
	}
	for _, share := range input.Shares {
		if !slices.Contains(userIds, share.UserID) {
			userIds = append(userIds, share.UserID)
		}
	}

	users, err := dbQuery.GetUsersByIds(ctx, userIds)
	if err != nil {
		return nil, err
	}
	if len(users) != len(userIds) {
		return nil, errorhelper.NewAppError(UserDoesNotExist, Messages[UserDoesNotExist], nil)
	}

	usersMap := make(map[int64]db.User)
	for _, user := range users {
		usersMap[user.ID] = user
	}

	return usersMap, nil
}

func createExpense(ctx context.Context, qtx *db.Queries, groupId int64, input model.ExpenseInput, currency db.Currency, usersMap map[int64]db.User) (*model.Expense, error) {
	expense, err := qtx.CreateExpense(ctx, db.CreateExpenseParams{
		GroupID:    groupId,
		Type:       ExpenseTypeGeneric,
		Amount:     pghelper.FromDecimal(input.Amount),
		CurrencyID: input.CurrencyID,
		ExpenseAt:  pghelper.Time(&input.ExpenseAt),
	})
	if err != nil {
		return nil, err
	}

	response := &model.Expense{
		ID:     expense.ID,
		Type:   expenseTypeString(expense.Type),
		Amount: pghelper.Decimal(expense.Amount),
		Currency: &model.Currency{
			ID:     currency.ID,
			Code:   currency.Code,
			Name:   currency.Name,
			Symbol: currency.Symbol,
		},
		ExpenseAt: expense.ExpenseAt.Time,
		Payers:    make([]*model.Share, 0),
		Shares:    make([]*model.Share, 0),
	}

	for _, payer := range input.Payers {
		expensePayer, err := qtx.CreateExpensePayer(ctx, db.CreateExpensePayerParams{
			ExpenseID: expense.ID,
			UserID:    payer.UserID,
			Amount:    pghelper.FromDecimal(payer.Amount),
		})
		if err != nil {
			return nil, err
		}

		response.Payers = append(response.Payers, &model.Share{
			User: &model.User{
				ID:   expensePayer.UserID,
				Name: usersMap[expensePayer.UserID].Username,
			},
			Amount: pghelper.Decimal(expensePayer.Amount),
		})
	}

	for _, share := range input.Shares {
		expenseSharer, err := qtx.CreateExpenseShare(ctx, db.CreateExpenseShareParams{
			ExpenseID: expense.ID,
			UserID:    share.UserID,
			Amount:    pghelper.FromDecimal(share.Amount),
		})
		if err != nil {
			return nil, err
		}

		response.Shares = append(response.Shares, &model.Share{
			User: &model.User{
				ID:   expenseSharer.UserID,
				Name: usersMap[expenseSharer.UserID].Username,
			},
			Amount: pghelper.Decimal(expenseSharer.Amount),
		})
	}
	return response, nil
}

type balance struct {
	UserID int64
	Amount decimal.Decimal
}

func sortBalanceAsc(a, b balance) int {
	cmp := a.Amount.Cmp(b.Amount)
	if cmp != 0 {
		return cmp
	}

	if a.UserID < b.UserID {
		return -1
	} else if a.UserID > b.UserID {
		return 1
	}
	return 0
}

func sortBalanceDesc(a, b balance) int {
	cmp := b.Amount.Cmp(a.Amount)
	if cmp != 0 {
		return cmp
	}

	if a.UserID < b.UserID {
		return -1
	} else if a.UserID > b.UserID {
		return 1
	}
	return 0
}
