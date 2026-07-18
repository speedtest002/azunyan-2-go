package db

import (
	"context"
	"database/sql"
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
		return 0, err
	}
	return newID, nil
}

func (d *DB) GetPendingUserReminders(ctx context.Context, from, to time.Time) ([]UserReminder, error) {
	userReminder := make([]UserReminder, 0, 10)

	rows, err := d.QueryContext(ctx, `
		SELECT ID, USER_ID, GUILD_ID, CHANNEL_ID, CONTENT, DUE_AT, STATUS, CREATED_AT, CLAIMED_AT
		FROM USER_REMINDER
		WHERE 	STATUS = 'PENDING'
			AND DUE_AT > :since AND DUE_AT <= :until
		ORDER BY DUE_AT
	`,
		sql.Named("since", from),
		sql.Named("until", to),
	)
	if err != nil {
		return userReminder, err
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
			return userReminder, err
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
		return userReminder, err
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
		return userReminder, err
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
			return userReminder, err
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
		return userReminder, err
	}

	return userReminder, nil
}

func (d *DB) ClaimUserReminder(ctx context.Context, id int64) (userID int64, message string, channelID int64, dueAt time.Time, claimed bool, err error) {
	res, err := d.ExecContext(ctx, `
        UPDATE USER_REMINDER
        SET STATUS = 'PROCESSING', CLAIMED_AT = SYSTIMESTAMP
        WHERE ID = :id AND STATUS = 'PENDING'
    `,
		sql.Named("id", id),
	)
	if err != nil {
		return 0, "", 0, time.Time{}, false, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, "", 0, time.Time{}, false, err
	}
	if rowsAffected == 0 {
		return 0, "", 0, time.Time{}, false, nil
	}

	err = d.QueryRowContext(ctx, `
        SELECT USER_ID, CONTENT, CHANNEL_ID, DUE_AT
        FROM USER_REMINDER
        WHERE ID = :id
    `,
		sql.Named("id", id),
	).Scan(&userID, &message, &channelID, &dueAt)
	if err != nil {
		return 0, "", 0, time.Time{}, false, err
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
	return err
}

func (d *DB) CancelUserReminder(ctx context.Context, reminderID, userID int64) error {
	_, err := d.ExecContext(ctx, `
		UPDATE USER_REMINDER
		SET STATUS = 'CANCELLED'
		WHERE ID = :id AND USER_ID = :userID
	`,
		sql.Named("id", reminderID),
		sql.Named("userID", userID),
	)
	return err
}
