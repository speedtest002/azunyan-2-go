package db

import (
	"context"
	"database/sql"
)

type UserBankInfo struct {
	UserID        int    `json:"user_id"`
	AccountNumber string `json:"account_number"`
	Bank          string `json:"bank"`
}

func (d *DB) UpsertUserBankInfo(ctx context.Context, userID int64, accountNumber, bank string) error {
	_, err := d.ExecContext(ctx, `
		MERGE INTO user_bank_info t
		USING (SELECT :user_id AS user_id, :account_number AS account_number, :bank AS bank FROM DUAL) s
		ON (t.user_id = s.user_id)
		WHEN MATCHED THEN UPDATE SET t.account_number = s.account_number, t.bank = s.bank
		WHEN NOT MATCHED THEN INSERT (user_id, account_number, bank) VALUES (s.user_id, s.account_number, s.bank)
	`,
		sql.Named("user_id", userID),
		sql.Named("account_number", accountNumber),
		sql.Named("bank", bank),
	)
	return err
}

func (d *DB) GetUserBankInfo(ctx context.Context, userID int64) (UserBankInfo, error) {
	var info UserBankInfo
	err := d.QueryRowContext(ctx, "SELECT user_id, account_number, bank FROM user_bank_info WHERE user_id = :user_id",
		sql.Named("user_id", userID),
	).Scan(&info.UserID, &info.AccountNumber, &info.Bank)
	if err == sql.ErrNoRows {
		return info, nil
	}
	return info, err
}
