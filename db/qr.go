package db

type UserBankInfo struct {
	UserID        int    `json:"user_id"`
	AccountNumber string `json:"account_number"`
	Bank          string `json:"bank"`
}

func (d *DB) UpsertUserBankInfo(userID int64, accountNumber, bank string) error {
	_, err := d.Exec(`
		MERGE INTO user_bank_info t
		USING (SELECT :1 AS user_id, :2 AS account_number, :3 AS bank FROM DUAL) s
		ON (t.user_id = s.user_id)
		WHEN MATCHED THEN UPDATE SET t.account_number = s.account_number, t.bank = s.bank
		WHEN NOT MATCHED THEN INSERT (user_id, account_number, bank) VALUES (s.user_id, s.account_number, s.bank)
	`, userID, accountNumber, bank)
	return err
}

func (d *DB) GetUserBankInfo(userID int64) (UserBankInfo, error) {
	var info UserBankInfo
	err := d.QueryRow("SELECT user_id, account_number, bank FROM user_bank_info WHERE user_id = :1", userID).
		Scan(&info.UserID, &info.AccountNumber, &info.Bank)
	return info, err
}
