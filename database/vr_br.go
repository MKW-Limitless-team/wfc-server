package database

import (
	"context"
	"errors"
	"time"
	"wwfc/common"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type VRBR struct {
	ProfileId uint32
	VR        int
	BR        int
	UpdatedAt time.Time
}

const (
	InsertVRBRQuery = `INSERT INTO vr_br (profile_id, vr, br) VALUES ($1, $2, $3)`
	UpdateVRBRQuery = `UPDATE vr_br SET vr = $2, br = $3, updated_at = CURRENT_TIMESTAMP WHERE profile_id = $1`
	GetVRBRQuery    = `SELECT vr, br FROM vr_br WHERE profile_id = $1`
)

func (c *Connection) GetVRBR(profileId uint32) (VRBR, bool) {
	vrbr := VRBR{ProfileId: profileId}
	row := c.pool.QueryRow(c.ctx, GetVRBRQuery, profileId)
	err := row.Scan(&vrbr.VR, &vrbr.BR)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return VRBR{}, false
		}
		panic(err)
	}
	return vrbr, true
}

func GetVRBRForPool(pool *pgxpool.Pool, ctx context.Context, profileId uint32) (VRBR, bool) {
	vrbr := VRBR{ProfileId: profileId}
	err := pool.QueryRow(ctx, GetVRBRQuery, profileId).Scan(&vrbr.VR, &vrbr.BR)
	if err != nil {
		return VRBR{}, false
	}
	return vrbr, true
}

func (c *Connection) SetVRBR(profileId uint32, vr int, br int) error {
	// Check if entry exists
	_, exists := c.GetVRBR(profileId)
	if exists {
		_, err := c.pool.Exec(c.ctx, UpdateVRBRQuery, profileId, vr, br)
		return err
	}
	_, err := c.pool.Exec(c.ctx, InsertVRBRQuery, profileId, vr, br)
	return err
}

func (c *Connection) InitializeVRBRForProfile(profileId uint32, defaultVR int, defaultBR int) error {
	// Initialise a new profile with the configured default VR/BR
	_, err := c.pool.Exec(c.ctx, InsertVRBRQuery, profileId, defaultVR, defaultBR)
	return err
}

// SetVRBRForPool sets the absolute VR/BR for a player, inserting a row if needed.
func SetVRBRForPool(pool *pgxpool.Pool, ctx context.Context, profileId uint32, vr int, br int) error {
	var exists int
	err := pool.QueryRow(ctx, `SELECT 1 FROM vr_br WHERE profile_id = $1`, profileId).Scan(&exists)
	if err == nil {
		_, err = pool.Exec(ctx, UpdateVRBRQuery, profileId, vr, br)
		return err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = pool.Exec(ctx, InsertVRBRQuery, profileId, vr, br)
		return err
	}
	return err
}

// ApplyVRDeltaForPool applies a VR delta to a player, clamped to [1, MaxVRLimit].
// Returns the new VR/BR so the caller can push the update to the client.
func ApplyVRDeltaForPool(pool *pgxpool.Pool, ctx context.Context, profileId uint32, vrDelta int) (int, int, error) {
	config := common.GetConfig()
	vrbr := config.VRBR

	var currentVR, currentBR int
	err := pool.QueryRow(ctx, GetVRBRQuery, profileId).Scan(&currentVR, &currentBR)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if _, err := pool.Exec(ctx, InsertVRBRQuery, profileId, vrbr.DefaultVR, vrbr.DefaultBR); err != nil {
				return 0, 0, err
			}
			currentVR = vrbr.DefaultVR
			currentBR = vrbr.DefaultBR
		} else {
			return 0, 0, err
		}
	}

	newVR := currentVR + vrDelta
	if newVR > vrbr.MaxVRLimit {
		newVR = vrbr.MaxVRLimit
	}
	if newVR < 1 {
		newVR = 1
	}

	if err := SetVRBRForPool(pool, ctx, profileId, newVR, currentBR); err != nil {
		return 0, 0, err
	}
	return newVR, currentBR, nil
}
