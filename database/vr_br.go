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
	GetAllVRBRQuery = `SELECT profile_id, vr, br, updated_at FROM vr_br`
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

func (c *Connection) GetAllVRBR() ([]VRBR, error) {
	rows, err := c.pool.Query(c.ctx, GetAllVRBRQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	vrbrList := make([]VRBR, 0)
	for rows.Next() {
		var vrbr VRBR
		err := rows.Scan(&vrbr.ProfileId, &vrbr.VR, &vrbr.BR, &vrbr.UpdatedAt)
		if err != nil {
			return nil, err
		}
		vrbrList = append(vrbrList, vrbr)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return vrbrList, nil
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

// ApplyVRBRChange applies a VR/BR delta to a player, enforcing the configured
// gain/loss limits, multipliers, and absolute maximum limits. The delta is
// clamped to the configured per-transaction limits before being applied.
func (c *Connection) ApplyVRBRChange(profileId uint32, vrDelta int, brDelta int) error {
	config := common.GetConfig()
	vrbr := config.VRBR

	// Clamp the deltas to the configured per-transaction limits.
	vrDelta = clampDelta(vrDelta, vrbr.MaxVRGain, vrbr.MaxVRLoss, vrbr.VRGainMultiplier, vrbr.VRLossMultiplier)
	brDelta = clampDelta(brDelta, vrbr.MaxBRGain, vrbr.MaxBRLoss, vrbr.BRGainMultiplier, vrbr.BRLossMultiplier)

	current, exists := c.GetVRBR(profileId)
	if !exists {
		// No entry yet; initialize with defaults then apply the delta.
		if err := c.InitializeVRBRForProfile(profileId, vrbr.DefaultVR, vrbr.DefaultBR); err != nil {
			return err
		}
		current, _ = c.GetVRBR(profileId)
	}

	newVR := current.VR + vrDelta
	newBR := current.BR + brDelta

	// Enforce the absolute maximum limits.
	if newVR > vrbr.MaxVRLimit {
		newVR = vrbr.MaxVRLimit
	}
	if newBR > vrbr.MaxBRLimit {
		newBR = vrbr.MaxBRLimit
	}
	if newVR < 0 {
		newVR = 0
	}
	if newBR < 0 {
		newBR = 0
	}

	return c.SetVRBR(profileId, newVR, newBR)
}

func clampDelta(delta int, maxGain int, maxLoss int, gainMultiplier float64, lossMultiplier float64) int {
	if delta > 0 {
		limit := int(float64(maxGain) * gainMultiplier)
		if delta > limit {
			return limit
		}
		return delta
	}
	limit := int(float64(maxLoss) * lossMultiplier)
	if delta < -limit {
		return -limit
	}
	return delta
}

func (c *Connection) InitializeVRBRForProfile(profileId uint32, defaultVR int, defaultBR int) error {
	// Initialize new profile with the configured default VR/BR
	_, err := c.pool.Exec(c.ctx, InsertVRBRQuery, profileId, defaultVR, defaultBR)
	return err
}

// ApplyVRBRChangeForPool applies a VR/BR delta to a player using a raw pool,
// enforcing the configured gain/loss limits, multipliers, and absolute maximum
// limits. This mirrors ApplyVRBRChange but works with a *pgxpool.Pool so it can
// be called from packages that only hold a pool reference (e.g. qr2). It returns
// the new VR/BR values so callers can push the update back to the client.
func ApplyVRBRChangeForPool(pool *pgxpool.Pool, ctx context.Context, profileId uint32, vrDelta int, brDelta int) (int, int, error) {
	config := common.GetConfig()
	vrbr := config.VRBR

	vrDelta = clampDelta(vrDelta, vrbr.MaxVRGain, vrbr.MaxVRLoss, vrbr.VRGainMultiplier, vrbr.VRLossMultiplier)
	brDelta = clampDelta(brDelta, vrbr.MaxBRGain, vrbr.MaxBRLoss, vrbr.BRGainMultiplier, vrbr.BRLossMultiplier)

	// Fetch current values.
	var currentVR, currentBR int
	err := pool.QueryRow(ctx, GetVRBRQuery, profileId).Scan(&currentVR, &currentBR)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No entry yet; initialize with defaults then apply the delta.
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
	newBR := currentBR + brDelta

	if newVR > vrbr.MaxVRLimit {
		newVR = vrbr.MaxVRLimit
	}
	if newBR > vrbr.MaxBRLimit {
		newBR = vrbr.MaxBRLimit
	}
	if newVR < 0 {
		newVR = 0
	}
	if newBR < 0 {
		newBR = 0
	}

	_, err = pool.Exec(ctx, UpdateVRBRQuery, profileId, newVR, newBR)
	return newVR, newBR, err
}
