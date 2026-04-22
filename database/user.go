package database

import (
	"errors"
	"math/rand"
	"time"
)

const (
	InsertUser              = `INSERT INTO users (user_id, gsbrcd, password, ng_device_id, email, unique_nick) VALUES ($1, $2, $3, $4, $5, $6) RETURNING profile_id`
	InsertUserWithProfileID = `INSERT INTO users (profile_id, user_id, gsbrcd, password, ng_device_id, email, unique_nick) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	UpdateUserTable         = `UPDATE users SET firstname = CASE WHEN $3 THEN $2 ELSE firstname END, lastname = CASE WHEN $5 THEN $4 ELSE lastname END, open_host = CASE WHEN $7 THEN $6 ELSE open_host END WHERE profile_id = $1`
	UpdateUserProfileID     = `UPDATE users SET profile_id = $3 WHERE user_id = $1 AND gsbrcd = $2`
	UpdateUserNGDeviceID    = `UPDATE users SET ng_device_id = $2 WHERE profile_id = $1`
	GetUser                 = `SELECT users.user_id, users.gsbrcd, users.ng_device_id, users.email, users.unique_nick, users.firstname, users.lastname, users.has_ban, users.ban_reason, users.open_host, users.last_ingamesn, users.last_ip_address, player_data.discord_id, users.ban_moderator, users.ban_reason_hidden, users.ban_issued, users.ban_expires FROM users LEFT JOIN player_data ON player_data.profile_id = users.profile_id WHERE users.profile_id = $1`
	ClearProfileQuery       = `DELETE FROM users WHERE profile_id = $1 RETURNING user_id, gsbrcd, email, unique_nick, firstname, lastname, open_host, last_ip_address, last_ingamesn`
	DoesUserExist           = `SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1 AND gsbrcd = $2)`
	IsProfileIDInUse        = `SELECT EXISTS(SELECT 1 FROM users WHERE profile_id = $1)`
	DeleteUserSession       = `DELETE FROM sessions WHERE profile_id = $1`
	GetUserProfileID        = `SELECT users.profile_id, users.ng_device_id, users.email, users.unique_nick, users.firstname, users.lastname, users.open_host, users.last_ip_address, users.allow_default_keys FROM users WHERE users.user_id = $1 AND users.gsbrcd = $2`
	UpdateUserLastIPAddress = `UPDATE users SET last_ip_address = $2, last_ingamesn = $3 WHERE profile_id = $1`
	UpdateUserBan           = `UPDATE users SET has_ban = true, ban_issued = $2, ban_expires = $3, ban_reason = $4, ban_reason_hidden = $5, ban_moderator = $6, ban_tos = $7 WHERE profile_id = $1`
	DisableUserBan          = `UPDATE users SET has_ban = false WHERE profile_id = $1`

	GetMKWFriendInfoQuery    = `SELECT mariokartwii_friend_info FROM users WHERE profile_id = $1`
	UpdateMKWFriendInfoQuery = `UPDATE users SET mariokartwii_friend_info = $2 WHERE profile_id = $1`
)

type User struct {
	ProfileId          uint32
	UserId             uint64
	GsbrCode           string
	NgDeviceId         []uint32
	Email              string
	UniqueNick         string
	FirstName          string
	LastName           string
	Restricted         bool
	RestrictedDeviceId uint32
	BanReason          string
	OpenHost           bool
	LastInGameSn       string
	LastIPAddress      string
	DiscordID          string
	BanModerator       string
	BanReasonHidden    string
	BanIssued          *time.Time
	BanExpires         *time.Time
	Created            bool
}

var (
	ErrProfileIDInUse         = errors.New("profile ID is already in use")
	ErrReservedProfileIDRange = errors.New("profile ID is in reserved range")
)

func (c *Connection) CreateUser(user *User) error {
	if user.ProfileId == 0 {
		return c.pool.QueryRow(c.ctx, InsertUser, user.UserId, user.GsbrCode, "", user.NgDeviceId, user.Email, user.UniqueNick).Scan(&user.ProfileId)
	}

	// Reserved profile ID check removed; all profile IDs allowed

	var exists bool
	err := c.pool.QueryRow(c.ctx, IsProfileIDInUse, user.ProfileId).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		return ErrProfileIDInUse
	}

	_, err = c.pool.Exec(c.ctx, InsertUserWithProfileID, user.ProfileId, user.UserId, user.GsbrCode, "", user.NgDeviceId, user.Email, user.UniqueNick)
	return err
}

func (c *Connection) UpdateProfileID(user *User, newProfileId uint32) error {
	// Reserved profile ID check removed; all profile IDs allowed

	var exists bool
	err := c.pool.QueryRow(c.ctx, IsProfileIDInUse, newProfileId).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		return ErrProfileIDInUse
	}

	_, err = c.pool.Exec(c.ctx, UpdateUserProfileID, user.UserId, user.GsbrCode, newProfileId)
	if err == nil {
		user.ProfileId = newProfileId
	}

	return err
}

func GetUniqueUserID() uint64 {
	// Not guaranteed unique but doesn't matter in practice if multiple people have the same user ID.
	return uint64(rand.Int63n(0x80000000000))
}

func (c *Connection) UpdateProfile(user *User, data map[string]string) {
	firstName, firstNameExists := data["firstname"]
	lastName, lastNameExists := data["lastname"]
	openHost, openHostExists := data["wl:oh"]
	openHostBool := openHostExists && openHost != "0"

	_, err := c.pool.Exec(c.ctx, UpdateUserTable, user.ProfileId, firstName, firstNameExists, lastName, lastNameExists, openHostBool, openHostExists)
	if err != nil {
		panic(err)
	}

	if firstNameExists {
		user.FirstName = firstName
	}

	if lastNameExists {
		user.LastName = lastName
	}

	if openHostExists {
		user.OpenHost = openHostBool
	}
}

func (c *Connection) getProfile(profileId uint32) (User, bool) {
	user := User{}
	var firstName *string
	var lastName *string
	var banReason *string
	var lastInGameSn *string
	var lastIPAddress *string
	var discordID *string
	var banModerator *string
	var banReasonHidden *string
	row := c.pool.QueryRow(c.ctx, GetUser, profileId)
	err := row.Scan(
		&user.UserId,
		&user.GsbrCode,
		&user.NgDeviceId,
		&user.Email,
		&user.UniqueNick,
		&firstName,
		&lastName,
		&user.Restricted,
		&banReason,
		&user.OpenHost,
		&lastInGameSn,
		&lastIPAddress,
		&discordID,
		&banModerator,
		&banReasonHidden,
		&user.BanIssued,
		&user.BanExpires,
	)
	if err != nil {
		return User{}, false
	}

	user.ProfileId = profileId

	if firstName != nil {
		user.FirstName = *firstName
	}

	if lastName != nil {
		user.LastName = *lastName
	}

	if banReason != nil {
		user.BanReason = *banReason
	}

	if lastInGameSn != nil {
		user.LastInGameSn = *lastInGameSn
	}

	if lastIPAddress != nil {
		user.LastIPAddress = *lastIPAddress
	}

	if discordID != nil {
		user.DiscordID = *discordID
	}

	if banModerator != nil {
		user.BanModerator = *banModerator
	}

	if banReasonHidden != nil {
		user.BanReasonHidden = *banReasonHidden
	}

	return user, true
}

func (c *Connection) ClearProfile(profileId uint32) (User, bool) {
	user := User{}
	row := c.pool.QueryRow(c.ctx, ClearProfileQuery, profileId)
	err := row.Scan(&user.UserId, &user.GsbrCode, &user.Email, &user.UniqueNick, &user.FirstName, &user.LastName, &user.OpenHost, &user.LastIPAddress, &user.LastInGameSn)

	if err != nil {
		return User{}, false
	}

	user.ProfileId = profileId
	return user, true
}

func (c *Connection) BanUser(profileId uint32, tos bool, length time.Duration, reason string, reasonHidden string, moderator string) bool {
	_, err := c.pool.Exec(c.ctx, UpdateUserBan, profileId, time.Now().UTC(), time.Now().UTC().Add(length), reason, reasonHidden, moderator, tos)
	return err == nil
}

func (c *Connection) UnbanUser(profileId uint32) bool {
	_, err := c.pool.Exec(c.ctx, DisableUserBan, profileId)
	return err == nil
}
