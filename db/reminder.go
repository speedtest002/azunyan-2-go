package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type UserReminder struct {
	ID        int64      `db:"ID"`
	UserID    int64      `db:"USER_ID"`
	GuildID   *int64     `db:"GUILD_ID"`
	ChannelID int64      `db:"CHANNEL_ID"`
	Content   string     `db:"CONTENT"`
	DueAt     time.Time  `db:"DUE_AT"`
	Status    string     `db:"STATUS"`
	CreatedAt time.Time  `db:"CREATED_AT"`
	ClaimedAt *time.Time `db:"CLAIMED_AT"`
}

type UserReminderPing struct {
	ReminderId int64 `db:"REMINDER_ID"`
	UserID     int64 `db:"USER_ID"`
}

// insert a reminder with status default "PENDING"
// return schedule id
func (d *DB) CreateUserReminder(ctx context.Context, userID, channelID int64, guildID *int64, content string, dueAt time.Time) (int64, error) {
	var newID int64

	_, err := d.ExecContext(ctx, `
		INSERT INTO USER_REMINDER
			(USER_ID, GUILD_ID, CHANNEL_ID, CONTENT, DUE_AT)
		VALUES (:user_id, :guild_id, :channel_id, :content, :due_at)
		RETURNING ID INTO :new_id
	`,
		sql.Named("user_id", userID),
		sql.Named("guild_id", guildID),
		sql.Named("channel_id", channelID),
		sql.Named("content", content),
		sql.Named("due_at", dueAt),
		sql.Named("new_id", sql.Out{Dest: &newID}),
	)
	if err != nil {
		return 0, fmt.Errorf("db.CreateUserReminder: %w", err)
	}
	return newID, nil
}

func (d *DB) GetPendingUserReminders(ctx context.Context, until time.Time) ([]UserReminder, error) {
	userReminder := make([]UserReminder, 0, 10)

	rows, err := d.QueryContext(ctx, `
		SELECT ID, USER_ID, GUILD_ID, CHANNEL_ID, CONTENT, DUE_AT, STATUS, CREATED_AT, CLAIMED_AT
		FROM USER_REMINDER
		WHERE 	STATUS = 'PENDING'
			AND DUE_AT <= :until
		ORDER BY DUE_AT
	`,
		sql.Named("until", until),
	)
	if err != nil {
		return userReminder, fmt.Errorf("db.GetPendingUserReminders query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var r UserReminder
		var guildID sql.NullInt64
		var claimedAt sql.NullTime

		err := rows.Scan(
			&r.ID,
			&r.UserID,
			&guildID,
			&r.ChannelID,
			&r.Content,
			&r.DueAt,
			&r.Status,
			&r.CreatedAt,
			&claimedAt,
		)
		if err != nil {
			return userReminder, fmt.Errorf("db.GetPendingUserReminders scan: %w", err)
		}

		if guildID.Valid {
			r.GuildID = &guildID.Int64
		}
		if claimedAt.Valid {
			r.ClaimedAt = &claimedAt.Time
		}
		userReminder = append(userReminder, r)
	}
	if err = rows.Err(); err != nil {
		return userReminder, fmt.Errorf("db.GetPendingUserReminders rows: %w", err)
	}

	return userReminder, nil
}

func (d *DB) GetUserReminders(ctx context.Context, userID int64, channelID *int64) ([]UserReminder, error) {
	userReminder := make([]UserReminder, 0, 10)

	rows, err := d.QueryContext(ctx, `
		SELECT ID, USER_ID, GUILD_ID, CHANNEL_ID, CONTENT, DUE_AT, STATUS, CREATED_AT, CLAIMED_AT
		FROM USER_REMINDER
		WHERE 	STATUS = 'PENDING'
			AND USER_ID = :user_id 
			AND (CHANNEL_ID = :channel_id OR :channel_id IS NULL)
		ORDER BY DUE_AT
	`,
		sql.Named("user_id", userID),
		sql.Named("channel_id", channelID),
	)
	if err != nil {
		return userReminder, fmt.Errorf("db.GetUserReminders query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var r UserReminder
		var guildID sql.NullInt64
		var claimedAt sql.NullTime

		err := rows.Scan(
			&r.ID,
			&r.UserID,
			&guildID,
			&r.ChannelID,
			&r.Content,
			&r.DueAt,
			&r.Status,
			&r.CreatedAt,
			&claimedAt,
		)
		if err != nil {
			return userReminder, fmt.Errorf("db.GetUserReminders scan: %w", err)
		}

		if guildID.Valid {
			r.GuildID = &guildID.Int64
		}
		if claimedAt.Valid {
			r.ClaimedAt = &claimedAt.Time
		}
		userReminder = append(userReminder, r)
	}
	if err = rows.Err(); err != nil {
		return userReminder, fmt.Errorf("db.GetUserReminders rows: %w", err)
	}

	return userReminder, nil
}

func (d *DB) ClaimUserReminder(ctx context.Context, reminderID int64) (userID int64, message string, channelID int64, dueAt time.Time, claimed bool, err error) {
	res, err := d.ExecContext(ctx, `
        UPDATE USER_REMINDER
        SET STATUS = 'PROCESSING', CLAIMED_AT = SYSTIMESTAMP
        WHERE ID = :id AND STATUS = 'PENDING'
    `,
		sql.Named("id", reminderID),
	)
	if err != nil {
		return 0, "", 0, time.Time{}, false, fmt.Errorf("db.ClaimUserReminder update: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, "", 0, time.Time{}, false, fmt.Errorf("db.ClaimUserReminder rowsAffected: %w", err)
	}
	if rowsAffected == 0 {
		return 0, "", 0, time.Time{}, false, nil
	}

	err = d.QueryRowContext(ctx, `
        SELECT USER_ID, CONTENT, CHANNEL_ID, DUE_AT
        FROM USER_REMINDER
        WHERE ID = :id
    `,
		sql.Named("id", reminderID),
	).Scan(&userID, &message, &channelID, &dueAt)
	if err != nil {
		return 0, "", 0, time.Time{}, false, fmt.Errorf("db.ClaimUserReminder select: %w", err)
	}

	return userID, message, channelID, dueAt, true, nil
}

// STATUS must be in ['PENDING', 'PROCESSING', 'SENT', 'FAILED', 'CANCELLED']
func (d *DB) UpdateStatusUserReminder(ctx context.Context, reminderID int64, status string) error {
	_, err := d.ExecContext(ctx, `
		UPDATE USER_REMINDER
		SET STATUS = :status
		WHERE ID = :id
	`,
		sql.Named("id", reminderID),
		sql.Named("status", status),
	)
	if err != nil {
		return fmt.Errorf("db.UpdateStatusUserReminder: %w", err)
	}
	return nil
}

func (d *DB) CancelUserReminder(ctx context.Context, reminderID, userID int64) (cancelled bool, err error) {
	res, err := d.ExecContext(ctx, `
		UPDATE USER_REMINDER
		SET STATUS = 'CANCELLED'
		WHERE ID = :id AND USER_ID = :userID AND STATUS = 'PENDING'
	`,
		sql.Named("id", reminderID),
		sql.Named("userID", userID),
	)
	if err != nil {
		return false, fmt.Errorf("db.CancelUserReminder: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("db.CancelUserReminder rowsAffected: %w", err)
	}

	return rowsAffected > 0, nil
}

func (d *DB) GetUsersReminderPing(ctx context.Context, reminderID int64) (userIDs []int64, err error) {
	rows, err := d.QueryContext(ctx, `
		SELECT USER_ID
		FROM USER_REMINDER_PING
		WHERE REMINDER_ID = :reminder_id
	`,
		sql.Named("reminder_id", reminderID),
	)

	if err != nil {
		return userIDs, fmt.Errorf("db.GetUsersReminderPing query: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var userID int64
		err := rows.Scan(&userID)
		if err != nil {
			return userIDs, fmt.Errorf("db.GetUsersReminderPing scan: %w", err)
		}
		userIDs = append(userIDs, userID)
	}

	if err = rows.Err(); err != nil {
		return userIDs, fmt.Errorf("db.GetUsersReminderPing rows: %w", err)
	}

	return userIDs, nil
}

func (d *DB) AddUserReminderPing(ctx context.Context, reminderID, userID int64) (added bool, err error) {
	res, err := d.ExecContext(ctx, `
		INSERT INTO USER_REMINDER_PING
			(REMINDER_ID, USER_ID)
		SELECT :reminder_id, :user_id FROM DUAL
		WHERE NOT EXISTS (
			SELECT 1 FROM USER_REMINDER_PING
			WHERE REMINDER_ID = :reminder_id AND USER_ID = :user_id
		)
		AND EXISTS (
			SELECT 1 FROM USER_REMINDER
			WHERE ID = :reminder_id AND STATUS = 'PENDING'
		)
	`,
		sql.Named("user_id", userID),
		sql.Named("reminder_id", reminderID),
	)
	if err != nil {
		return false, fmt.Errorf("db.AddUserReminderPing: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("db.AddUserReminderPing rowsAffected: %w", err)
	}

	return rowsAffected > 0, nil
}

func (d *DB) RemoveUserReminderPing(ctx context.Context, reminderID, userID int64) (removed bool, err error) {
	res, err := d.ExecContext(ctx, `
		DELETE FROM USER_REMINDER_PING
		WHERE REMINDER_ID = :reminder_id AND USER_ID = :user_id
		AND EXISTS (
			SELECT 1 FROM USER_REMINDER
			WHERE ID = :reminder_id AND STATUS = 'PENDING'
		)
	`,
		sql.Named("user_id", userID),
		sql.Named("reminder_id", reminderID),
	)
	if err != nil {
		return false, fmt.Errorf("db.RemoveUserReminderPing: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("db.RemoveUserReminderPing rowsAffected: %w", err)
	}

	return rowsAffected > 0, nil
}
