package gpcm

import (
	"encoding/binary"
	"strconv"
	"wwfc/common"
	"wwfc/logging"
	"wwfc/qr2"

	"github.com/logrusorgru/aurora/v3"
)

type RaceResultPlayer struct {
	Pid          int `json:"pid"`
	FinishTimeMs int `json:"finish_time_ms"`
	CharacterId  int `json:"character_id"`
	KartId       int `json:"kart_id"`
	PlayerCount  int `json:"player_count"`
}

type RaceResult struct {
	ClientReportVersion string            `json:"client_report_version"`
	Player              *RaceResultPlayer `json:"player"`
}

func (g *GameSpySession) handleWWFCReport(command common.GameSpyCommand) {
	for key, value := range command.OtherValues {
		logging.Info(g.ModuleName, "WiiLink Report:", aurora.Yellow(key))

		keyColored := aurora.BrightCyan(key).String()

		switch key {
		default:
			logging.Error(g.ModuleName, "Unknown record", aurora.Cyan(key).String()+":", aurora.Cyan(value))

		case "wl:bad_packet":
			profileId, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				logging.Error(g.ModuleName, "Error decoding", keyColored+":", err.Error())
				continue
			}

			logging.Warn(g.ModuleName, "Report bad packet from", aurora.BrightCyan(strconv.FormatUint(profileId, 10)))
			logging.Event("reported_bad_packet", map[string]any{
				"profile_id": g.User.ProfileId,
				"sender_id":  profileId,
			})

		case "wl:bad_packet_data":
			// Diagnostic record with the raw bytes of a packet that failed
			// validation on the client.
			logging.Warn(g.ModuleName, "Report bad packet data from", aurora.BrightCyan(strconv.FormatUint(uint64(g.User.ProfileId), 10)))
			logging.Event("reported_bad_packet_data", map[string]any{
				"profile_id": g.User.ProfileId,
				"data":       value,
			})

		case "wl:stall":
			profileId, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				logging.Error(g.ModuleName, "Error decoding", keyColored+":", err.Error())
				continue
			}

			logging.Warn(g.ModuleName, "Room stall caused by", aurora.BrightCyan(strconv.FormatUint(profileId, 10)))
			logging.Event("reported_stall", map[string]any{
				"profile_id":  g.User.ProfileId,
				"stalling_id": profileId,
			})

		case "wl:mkw_user":
			if g.GameName != "mariokartwii" {
				logging.Warn(g.ModuleName, "Ignoring", keyColored+":", "from wrong game")
				continue
			}

			packet, err := common.Base64DwcEncoding.DecodeString(value)
			if err != nil {
				logging.Error(g.ModuleName, "Error decoding", keyColored+":", err.Error())
				continue
			}

			if len(packet) != 0xC0 {
				logging.Error(g.ModuleName, "Invalid", keyColored, "record length:", len(packet))
				continue
			}

			qr2.ProcessUSER(g.User.ProfileId, g.QR2IP, packet)

		case "wl:mkw_select_course", "wl:mkw_select_cc":
			if g.GameName != "mariokartwii" {
				logging.Warn(g.ModuleName, "Ignoring", keyColored, "from wrong game")
				continue
			}

			qr2.ProcessMKWSelectRecord(g.User.ProfileId, key, value)

		case "wl:mkw_finish_time":
			if g.GameName != "mariokartwii" {
				logging.Warn(g.ModuleName, "Ignoring", keyColored, "from wrong game")
				continue
			}

			packet, err := common.Base64DwcEncoding.DecodeString(value)
			if err != nil {
				logging.Error(g.ModuleName, "Error decoding finish time:", err.Error())
				continue
			}
			if len(packet) != 12 {
				logging.Error(g.ModuleName, "Invalid finish time record length:", len(packet))
				continue
			}

			// The 12-byte report holds inGameTime, finishTime, and the lag difference.
			inGameTime := binary.BigEndian.Uint32(packet[0:4])
			finishTime := binary.BigEndian.Uint32(packet[4:8])
			delta := int32(binary.BigEndian.Uint32(packet[8:12]))
			logging.Info(g.ModuleName, "Received finish time", aurora.BrightCyan(strconv.FormatUint(uint64(finishTime), 10)),
				"in-game", aurora.BrightCyan(strconv.FormatUint(uint64(inGameTime), 10)),
				"lag", aurora.BrightCyan(strconv.FormatInt(int64(delta), 10)),
				"from profile", aurora.BrightCyan(strconv.FormatUint(uint64(g.User.ProfileId), 10)))

			ratings := qr2.ProcessMKWFinishTime(g.User.ProfileId, finishTime, delta)
			for pid, vrbr := range ratings {
				SendVRBRUpdate(pid, vrbr.VR, vrbr.BR)
			}
		}
	}
}

func SendVRBRUpdate(profileId uint32, vr int, br int) {
	mutex.Lock()
	session := sessions[profileId]
	mutex.Unlock()
	if session == nil {
		return
	}
	msg := "\\wl:vr\\" + strconv.Itoa(vr) + "\\wl:br\\" + strconv.Itoa(br) + "\\final\\"
	if err := common.SendPacket(ServerName, session.ConnIndex, []byte(msg)); err != nil {
		logging.Error(session.ModuleName, "Failed to send VR/BR update to client:", err)
	} else {
		logging.Info(session.ModuleName, "Sent VR/BR update to client:", vr, br)
	}
}

func init() {
	qr2.OnRaceFinalized = func(ratings map[uint32]qr2.VRBRResult) {
		for pid, vrbr := range ratings {
			SendVRBRUpdate(pid, vrbr.VR, vrbr.BR)
		}
	}
}
