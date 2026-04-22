package database

import (
	"context"
	"fmt"
	"wwfc/common"

	"github.com/jackc/pgx/v4/pgxpool"
)

type Connection struct {
	pool *pgxpool.Pool
	ctx  context.Context
}

func Start(config common.Config) Connection {
	conn := Connection{
		ctx: context.Background(),
	}

	dbString := fmt.Sprintf("postgres://%s:%s@%s/%s", config.Username, config.Password, config.DatabaseAddress, config.DatabaseName)
	dbConf, err := pgxpool.ParseConfig(dbString)
	if err != nil {
		panic(err)
	}

	conn.pool, err = pgxpool.ConnectConfig(conn.ctx, dbConf)
	if err != nil {
		panic(err)
	}

	return conn
}

func (c *Connection) Close() {
	if c != nil && c.pool != nil {
		c.pool.Close()
	}
}

func (c *Connection) GetProfile(profileId uint32) (User, bool) {
	return c.getProfile(profileId)
}

func (c *Connection) GetAllTrackFrequencies() ([]TrackEntry, error) {
	return GetAllTrackFrequencies(c.pool, c.ctx)
}

func (c *Connection) GetPlayerCharacterUsage(profileId uint32) (CharacterUsageEntry, error) {
	return GetPlayerCharacterUsage(c.pool, c.ctx, profileId)
}

func (c *Connection) GetPlayerVehicleUsage(profileId uint32) (VehicleUsageEntry, error) {
	return GetPlayerVehicleUsage(c.pool, c.ctx, profileId)
}
