package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type UserConfig struct {
	UserID    int64     `db:"USER_ID"`
	Timezone  string    `db:"TIMEZONE"` // IANA name, ex: "Asia/Ho_Chi_Minh"
	CreatedAt time.Time `db:"CREATED_AT"`
	UpdatedAt time.Time `db:"UPDATED_AT"`
}

// Upsert User's Timezone
func (d *DB) SetUserTimezone(ctx context.Context, userID int64, timezone string) error {
	if _, err := time.LoadLocation(timezone); err != nil {
		return fmt.Errorf("timezone không hợp lệ %q: %w", timezone, err)
	}

	_, err := d.ExecContext(ctx, `
		MERGE INTO USER_CONFIG dst
		USING (SELECT :user_id AS USER_ID FROM dual) src
		ON (dst.USER_ID = src.USER_ID)
		WHEN MATCHED THEN
			UPDATE SET dst.TIMEZONE = :timezone, dst.UPDATED_AT = SYSTIMESTAMP
		WHEN NOT MATCHED THEN
			INSERT (USER_ID, TIMEZONE)
			VALUES (:user_id, :timezone)
	`,
		sql.Named("user_id", userID),
		sql.Named("timezone", timezone),
	)
	if err != nil {
		return fmt.Errorf("db.SetUserTimezone: %w", err)
	}
	return nil
}

func (d *DB) GetUserLocation(ctx context.Context, userID int64, defaultTimezone *time.Location) (*time.Location, error) {
	var tz string
	err := d.QueryRowContext(ctx, `
		SELECT TIMEZONE FROM USER_CONFIG WHERE USER_ID = :user_id
	`, sql.Named("user_id", userID)).Scan(&tz)

	if defaultTimezone == nil {
		defaultTimezone, _ = time.LoadLocation("Asia/Ho_Chi_Minh")
	}

	if err == sql.ErrNoRows {
		return defaultTimezone, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db.GetUserLocation: %w", err)
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		return defaultTimezone, nil
	}
	return loc, nil
}
