package database

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v4/pgxpool"
)

//
// Track, Character, and Vehicle tracking functions
//

const (
	incrementTrackFrequencyQuery = "" +
		"INSERT INTO tracks (track_name, frequency) " +
		"VALUES ($1, 1) " +
		"ON CONFLICT (track_name) DO UPDATE " +
		"SET frequency = tracks.frequency + 1"

	getTrackFrequencyQuery = "" +
		"SELECT track_name, frequency " +
		"FROM tracks " +
		"ORDER BY frequency DESC"

	getCharacterUsageQuery = "" +
		"SELECT profile_id FROM characters " +
		"ORDER BY %s DESC " +
		"LIMIT $1 OFFSET $2"

	getVehicleUsageQuery = "" +
		"SELECT profile_id FROM vehicles " +
		"ORDER BY %s DESC " +
		"LIMIT $1 OFFSET $2"

	getPlayerCharacterUsageQuery = "" +
		"SELECT * FROM characters WHERE profile_id = $1"

	getPlayerVehicleUsageQuery = "" +
		"SELECT * FROM vehicles WHERE profile_id = $1"

	getGlobalCharacterUsageQuery = "" +
		"SELECT profile_id, %s FROM characters " +
		"ORDER BY %s DESC " +
		"LIMIT $1 OFFSET $2"

	getGlobalVehicleUsageQuery = "" +
		"SELECT profile_id, %s FROM vehicles " +
		"ORDER BY %s DESC " +
		"LIMIT $1 OFFSET $2"

	// Dynamic upsert queries using column name parameter
	incrementCharacterUsageQueryTemplate = "" +
		"INSERT INTO characters (profile_id, %s) " +
		"VALUES ($1, 1) " +
		"ON CONFLICT (profile_id) DO UPDATE " +
		"SET %s = characters.%s + 1"

	incrementVehicleUsageQueryTemplate = "" +
		"INSERT INTO vehicles (profile_id, %s) " +
		"VALUES ($1, 1) " +
		"ON CONFLICT (profile_id) DO UPDATE " +
		"SET %s = vehicles.%s + 1"
)

// CourseIDToName converts a numeric course ID to a human-readable track name.
func CourseIDToName(courseId int) string {
	ctMap := map[int]string{
		256: "Wii Mushroom Gorge",
		257: "Tour Choco Mountain",
		258: "DS Luigi's Mansion",
		259: "Wii Daisy Circuit",
		260: "SRB2K Green Hill Zone",
		261: "Goomba Circuit",
		262: "Eventide Gorge",
		263: "Toad's Temple",
		264: "DS DK Pass",
		265: "Wii Koopa Cape",
		266: "GBA Bowser Castle 2",
		267: "GCN Dino Dino Jungle",
		268: "Musical Cliff",
		269: "Desert Fort",
		270: "Camp Kartigan",
		271: "Fort Francis",
		272: "GCN Waluigi Stadium",
		273: "DS Peach Gardens",
		274: "Wii Grumble Volcano",
		275: "SNES Bowser Castle 2",
		276: "Forest Creek",
		277: "Tau-Cryovolcano",
		278: "Phendrana Frostbite",
		279: "Siberian Chateau",
		280: "N64 Sherbet Land",
		281: "Wii DK Summit",
		282: "GBA Snow Land",
		283: "3DS DK Jungle",
		284: "Super Marine World",
		285: "Frantic Funyard",
		286: "Honeybee Hideout",
		287: "Stickerbush Serenity",
		288: "GCN Daisy Cruiser",
		289: "N64 Banshee Boardwalk",
		290: "DS Waluigi Pinball",
		291: "Wii Wario's Gold Mine",
		292: "Coin Heaven",
		293: "The Grand Archives",
		294: "Midnight Museum",
		295: "Shy Guy Lumber Co.",
		296: "SNES Choco Island 2",
		297: "Wii Coconut Mall",
		298: "GBA Bowser Castle 3",
		299: "GCN Yoshi Circuit",
		300: "Fungal Jungle",
		301: "Ice Cream Fortress",
		302: "Kitayama Keep",
		303: "Bowser Jr.'s Crafty Castle",
		304: "DS Wario Stadium",
		305: "Wii Toad's Factory",
		306: "SW2 Great Question Block Ruins",
		307: "N64 Bowser's Castle",
		308: "BKMR Moai Mountain",
		309: "Terra Ursae",
		310: "Darkness Temple",
		311: "Ghost House",
		312: "3DS Music Park",
		313: "Wii Moonview Highway",
		314: "DS Airship Fortress",
		315: "N64 Wario Stadium",
		316: "Bluster Blob Bluff",
		317: "Poisonous Pass",
		318: "The Great Apple War",
		319: "Ruinated Peach's Castle",
		320: "N64 DK's Jungle Parkway",
		321: "DS Delfino Square",
		322: "Wii Maple Treeway",
		323: "Tour Bowser Castle 3",
		324: "Sandstone Cliffs",
		325: "Windy Whirl",
		326: "Quaking Mad Cliffs",
		327: "Castle in the Sky",
		328: "Wii Dry Dry Ruins",
		329: "DS Tick-Tock Clock",
		330: "GBA Bowser Castle 4",
		331: "SNES Rainbow Road",
		332: "Banished Keep",
		333: "Thump Bump Forest",
		334: "Obstagoon's Palace",
		335: "Cruel Angel's Thesis",
		336: "3DS Wario Shipyard",
		337: "GBA Boo Lake",
		338: "Wii Bowser's Castle",
		339: "N64 Rainbow Road",
		340: "Ghostly Gulch",
		341: "Spooky Swamp",
		342: "Northern Heights",
		343: "Vile Isle",
		344: "N64 Frappe Snowland",
		345: "GCN Sherbet Land",
		346: "GBA Sky Garden",
		347: "Wii Rainbow Road",
		348: "Cosmic Causeway",
		349: "Aqua Dungeon",
		350: "Garden of Dreams",
		351: "Bowser's Termination Station",
	}
	if name, ok := ctMap[courseId]; ok {
		return name
	}
	return strconv.Itoa(courseId)
}

// IncrementTrackFrequency increments the usage counter for a track by name.
// If the track doesn't exist, it creates a new entry with frequency 1.
func IncrementTrackFrequency(pool *pgxpool.Pool, ctx context.Context, trackName string) error {
	_, err := pool.Exec(ctx, incrementTrackFrequencyQuery, trackName)
	return err
}

// TrackEntry represents a single track frequency record
type TrackEntry struct {
	TrackName string `json:"track_name"`
	Frequency int64  `json:"frequency"`
}

// GetAllTrackFrequencies retrieves all tracks ordered by frequency descending
func GetAllTrackFrequencies(pool *pgxpool.Pool, ctx context.Context) ([]TrackEntry, error) {
	rows, err := pool.Query(ctx, getTrackFrequencyQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []TrackEntry
	for rows.Next() {
		var entry TrackEntry
		if err := rows.Scan(&entry.TrackName, &entry.Frequency); err != nil {
			return nil, err
		}
		tracks = append(tracks, entry)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tracks, nil
}

// IncrementCharacterUsage increments the usage counter for a character by player profile.
// Uses a placeholder column name - this should be updated with actual mappings.
// TODO: Replace with correct character ID to column name mapping
func IncrementCharacterUsage(pool *pgxpool.Pool, ctx context.Context, profileId uint32, characterId int) error {
	columnName, err := CharacterIDToColumn(characterId)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(incrementCharacterUsageQueryTemplate, columnName, columnName, columnName)
	_, err = pool.Exec(ctx, query, profileId)
	return err
}

// IncrementVehicleUsage increments the usage counter for a vehicle by player profile.
// Uses a placeholder column name - this should be updated with actual mappings.
// TODO: Replace with correct vehicle ID to column name mapping
func IncrementVehicleUsage(pool *pgxpool.Pool, ctx context.Context, profileId uint32, vehicleId int) error {
	columnName, err := VehicleIDToColumn(vehicleId)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(incrementVehicleUsageQueryTemplate, columnName, columnName, columnName)
	_, err = pool.Exec(ctx, query, profileId)
	return err
}

// CharacterUsageEntry represents a character usage record for a player
type CharacterUsageEntry struct {
	ProfileID   uint32 `json:"profile_id"`
	BabyDaisy   uint64 `json:"baby_daisy"`
	BabyLuigi   uint64 `json:"baby_luigi"`
	BabyMario   uint64 `json:"baby_mario"`
	BabyPeach   uint64 `json:"baby_peach"`
	Birdo       uint64 `json:"birdo"`
	Bowser      uint64 `json:"bowser"`
	BowserJr    uint64 `json:"bowser_jr"`
	Daisy       uint64 `json:"daisy"`
	DiddyKong   uint64 `json:"diddy_kong"`
	DonkeyKong  uint64 `json:"donkey_kong"`
	DryBones    uint64 `json:"dry_bones"`
	DryBowser   uint64 `json:"dry_bowser"`
	FunkyKong   uint64 `json:"funky_kong"`
	KingBoo     uint64 `json:"king_boo"`
	KoopaTroopa uint64 `json:"koopa_troopa"`
	Luigi       uint64 `json:"luigi"`
	Mario       uint64 `json:"mario"`
	MiiLAFemale uint64 `json:"mii_l_a_female"`
	MiiLAMale   uint64 `json:"mii_l_a_male"`
	MiiLBFemale uint64 `json:"mii_l_b_female"`
	MiiLBMale   uint64 `json:"mii_l_b_male"`
	MiiLCFemale uint64 `json:"mii_l_c_female"`
	MiiLCMale   uint64 `json:"mii_l_c_male"`
	MiiLarge    uint64 `json:"mii_large"`
	MiiMAFemale uint64 `json:"mii_m_a_female"`
	MiiMAMale   uint64 `json:"mii_m_a_male"`
	MiiMBFemale uint64 `json:"mii_m_b_female"`
	MiiMBMale   uint64 `json:"mii_m_b_male"`
	MiiMCFemale uint64 `json:"mii_m_c_female"`
	MiiMCMale   uint64 `json:"mii_m_c_male"`
	MiiMedium   uint64 `json:"mii_medium"`
	MiiSAFemale uint64 `json:"mii_s_a_female"`
	MiiSAMale   uint64 `json:"mii_s_a_male"`
	MiiSBFemale uint64 `json:"mii_s_b_female"`
	MiiSBMale   uint64 `json:"mii_s_b_male"`
	MiiSCFemale uint64 `json:"mii_s_c_female"`
	MiiSCMale   uint64 `json:"mii_s_c_male"`
	MiiSmall    uint64 `json:"mii_small"`
	Peach       uint64 `json:"peach"`
	Rosalina    uint64 `json:"rosalina"`
	Toad        uint64 `json:"toad"`
	Toadette    uint64 `json:"toadette"`
	Wario       uint64 `json:"wario"`
	Waluigi     uint64 `json:"waluigi"`
	Yoshi       uint64 `json:"yoshi"`
}

// GetPlayerCharacterUsage retrieves all character usage data for a specific player
func GetPlayerCharacterUsage(pool *pgxpool.Pool, ctx context.Context, profileId uint32) (CharacterUsageEntry, error) {
	var entry CharacterUsageEntry
	err := pool.QueryRow(ctx, getPlayerCharacterUsageQuery, profileId).Scan(
		&entry.ProfileID,
		&entry.BabyDaisy,
		&entry.BabyLuigi,
		&entry.BabyMario,
		&entry.BabyPeach,
		&entry.Birdo,
		&entry.Bowser,
		&entry.BowserJr,
		&entry.Daisy,
		&entry.DiddyKong,
		&entry.DonkeyKong,
		&entry.DryBones,
		&entry.DryBowser,
		&entry.FunkyKong,
		&entry.KingBoo,
		&entry.KoopaTroopa,
		&entry.Luigi,
		&entry.Mario,
		&entry.MiiLAFemale,
		&entry.MiiLAMale,
		&entry.MiiLBFemale,
		&entry.MiiLBMale,
		&entry.MiiLCFemale,
		&entry.MiiLCMale,
		&entry.MiiLarge,
		&entry.MiiMAFemale,
		&entry.MiiMAMale,
		&entry.MiiMBFemale,
		&entry.MiiMBMale,
		&entry.MiiMCFemale,
		&entry.MiiMCMale,
		&entry.MiiMedium,
		&entry.MiiSAFemale,
		&entry.MiiSAMale,
		&entry.MiiSBFemale,
		&entry.MiiSBMale,
		&entry.MiiSCFemale,
		&entry.MiiSCMale,
		&entry.MiiSmall,
		&entry.Peach,
		&entry.Rosalina,
		&entry.Toad,
		&entry.Toadette,
		&entry.Wario,
		&entry.Waluigi,
		&entry.Yoshi,
	)
	return entry, err
}

// VehicleUsageEntry represents a vehicle usage record for a player
type VehicleUsageEntry struct {
	ProfileID       uint64 `json:"profile_id"`
	BitBike         uint64 `json:"bit_bike"`
	BlueFalcon      uint64 `json:"blue_falcon"`
	BoosterSeat     uint64 `json:"booster_seat"`
	BulletBike      uint64 `json:"bullet_bike"`
	CheepCharger    uint64 `json:"cheep_charger"`
	ClassicDragster uint64 `json:"classic_dragster"`
	Daytripper      uint64 `json:"daytripper"`
	DolphinDasher   uint64 `json:"dolphin_dasher"`
	FlameFlyer      uint64 `json:"flame_flyer"`
	FlameRunner     uint64 `json:"flame_runner"`
	Honeycoupe      uint64 `json:"honeycoupe"`
	JetBubble       uint64 `json:"jet_bubble"`
	Jetsetter       uint64 `json:"jetsetter"`
	MachBike        uint64 `json:"mach_bike"`
	Magikruiser     uint64 `json:"magikruiser"`
	MiniBeast       uint64 `json:"mini_beast"`
	Offroader       uint64 `json:"offroader"`
	Phantom         uint64 `json:"phantom"`
	PiranhaProwler  uint64 `json:"piranha_prowler"`
	Quacker         uint64 `json:"quacker"`
	ShootingStar    uint64 `json:"shooting_star"`
	Sneakster       uint64 `json:"sneakster"`
	Spear           uint64 `json:"spear"`
	Sprinter        uint64 `json:"sprinter"`
	StandardBikeL   uint64 `json:"standard_bike_l"`
	StandardBikeM   uint64 `json:"standard_bike_m"`
	StandardBikeS   uint64 `json:"standard_bike_s"`
	StandardKartL   uint64 `json:"standard_kart_l"`
	StandardKartM   uint64 `json:"standard_kart_m"`
	StandardKartS   uint64 `json:"standard_kart_s"`
	Sugarscoot      uint64 `json:"sugarscoot"`
	SuperBlooper    uint64 `json:"super_blooper"`
	TinyTitan       uint64 `json:"tiny_titan"`
	WarioBike       uint64 `json:"wario_bike"`
	WildWing        uint64 `json:"wild_wing"`
	ZipZip          uint64 `json:"zip_zip"`
}

// GetPlayerVehicleUsage retrieves all vehicle usage data for a specific player
func GetPlayerVehicleUsage(pool *pgxpool.Pool, ctx context.Context, profileId uint32) (VehicleUsageEntry, error) {
	var entry VehicleUsageEntry
	err := pool.QueryRow(ctx, getPlayerVehicleUsageQuery, profileId).Scan(
		&entry.ProfileID,
		&entry.BitBike,
		&entry.BlueFalcon,
		&entry.BoosterSeat,
		&entry.BulletBike,
		&entry.CheepCharger,
		&entry.ClassicDragster,
		&entry.Daytripper,
		&entry.DolphinDasher,
		&entry.FlameFlyer,
		&entry.FlameRunner,
		&entry.Honeycoupe,
		&entry.JetBubble,
		&entry.Jetsetter,
		&entry.MachBike,
		&entry.Magikruiser,
		&entry.MiniBeast,
		&entry.Offroader,
		&entry.Phantom,
		&entry.PiranhaProwler,
		&entry.Quacker,
		&entry.ShootingStar,
		&entry.Sneakster,
		&entry.Spear,
		&entry.Sprinter,
		&entry.StandardBikeL,
		&entry.StandardBikeM,
		&entry.StandardBikeS,
		&entry.StandardKartL,
		&entry.StandardKartM,
		&entry.StandardKartS,
		&entry.Sugarscoot,
		&entry.SuperBlooper,
		&entry.TinyTitan,
		&entry.WarioBike,
		&entry.WildWing,
		&entry.ZipZip,
	)
	return entry, err
}

// CharacterIDToColumn converts a Mario Kart Wii character ID to the corresponding database column name.
func CharacterIDToColumn(characterId int) (string, error) {
	characterMapping := map[int]string{
		0:  "mario",
		1:  "baby_peach",
		2:  "waluigi",
		3:  "bowser",
		4:  "baby_daisy",
		5:  "dry_bones",
		6:  "baby_mario",
		7:  "luigi",
		8:  "toad",
		9:  "donkey_kong",
		10: "yoshi",
		11: "wario",
		12: "baby_luigi",
		13: "toadette",
		14: "koopa_troopa",
		15: "daisy",
		16: "peach",
		17: "birdo",
		18: "diddy_kong",
		19: "king_boo",
		20: "bowser_jr",
		21: "dry_bowser",
		22: "funky_kong",
		23: "rosalina",
		24: "mii_s_a_male",
		25: "mii_s_a_female",
		26: "mii_s_b_male",
		27: "mii_s_b_female",
		28: "mii_s_c_male",
		29: "mii_s_c_female",
		30: "mii_m_a_male",
		31: "mii_m_a_female",
		32: "mii_m_b_male",
		33: "mii_m_b_female",
		34: "mii_m_c_male",
		35: "mii_m_c_female",
		36: "mii_l_a_male",
		37: "mii_l_a_female",
		38: "mii_l_b_male",
		39: "mii_l_b_female",
		40: "mii_l_c_male",
		41: "mii_l_c_female",
	}

	if column, ok := characterMapping[characterId]; ok {
		return column, nil
	}

	return "", fmt.Errorf("unknown character ID: %d", characterId)
}

// VehicleIDToColumn converts a Mario Kart Wii vehicle ID to the corresponding database column name.
func VehicleIDToColumn(vehicleId int) (string, error) {
	vehicleMapping := map[int]string{
		0:  "standard_kart_s",
		1:  "standard_kart_m",
		2:  "standard_kart_l",
		3:  "booster_seat",
		4:  "classic_dragster",
		5:  "offroader",
		6:  "mini_beast",
		7:  "wild_wing",
		8:  "flame_flyer",
		9:  "cheep_charger",
		10: "super_blooper",
		11: "piranha_prowler",
		12: "tiny_titan",
		13: "daytripper",
		14: "jetsetter",
		15: "blue_falcon",
		16: "sprinter",
		17: "honeycoupe",
		18: "standard_bike_s",
		19: "standard_bike_m",
		20: "standard_bike_l",
		21: "bullet_bike",
		22: "mach_bike",
		23: "flame_runner",
		24: "bit_bike",
		25: "sugarscoot",
		26: "wario_bike",
		27: "quacker",
		28: "zip_zip",
		29: "shooting_star",
		30: "magikruiser",
		31: "sneakster",
		32: "spear",
		33: "jet_bubble",
		34: "dolphin_dasher",
		35: "phantom",
	}

	if column, ok := vehicleMapping[vehicleId]; ok {
		return column, nil
	}

	return "", fmt.Errorf("unknown vehicle ID: %d", vehicleId)
}

// IncrementTrackUsageForProfile increments all relevant character and vehicle usage counters
// for a player based on race result data.
func IncrementTrackUsageForProfile(pool *pgxpool.Pool, ctx context.Context, profileId uint32, characterId int, vehicleId int) error {
	if characterId >= 0 {
		if err := IncrementCharacterUsage(pool, ctx, profileId, characterId); err != nil {
			return fmt.Errorf("failed to increment character usage: %w", err)
		}
	}

	if vehicleId >= 0 {
		if err := IncrementVehicleUsage(pool, ctx, profileId, vehicleId); err != nil {
			return fmt.Errorf("failed to increment vehicle usage: %w", err)
		}
	}

	return nil
}

// SelectTrackUsageForProfile increments the track frequency for a given course.
func SelectTrackUsageForProfile(pool *pgxpool.Pool, ctx context.Context, courseId int) error {
	if courseId >= 0 {
		trackName := CourseIDToName(courseId)
		if err := IncrementTrackFrequency(pool, ctx, trackName); err != nil {
			return fmt.Errorf("failed to increment track frequency: %w", err)
		}
	}
	return nil
}