package api

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"
)

type PinfoRequestSpec struct {
	Secret    string `json:"secret"`
	ProfileID uint32 `json:"pid"`
}

type PinfoResponse struct {
	Player  PinfoPlayer `json:"player"`
	Success bool        `json:"success"`
	Error   string      `json:"error"`
}

type PinfoPlayer struct {
	ProfileID uint32 `json:"profile_id"`
	MiiName   string `json:"mii_name"`
	MiiData   string `json:"mii_data"`
	OpenHost  bool   `json:"open_host"`
	Banned    bool   `json:"banned"`
	DiscordID string `json:"discord_id"`
}

func HandlePinfo(w http.ResponseWriter, r *http.Request) {
	var response PinfoResponse
	var statusCode int

	switch r.Method {
	case http.MethodPost:
		response, statusCode = handlePinfoImpl(r)
	case http.MethodOptions:
		statusCode = http.StatusNoContent
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	default:
		statusCode = http.StatusMethodNotAllowed
		w.Header().Set("Allow", "POST")
		response = PinfoResponse{
			Success: false,
			Error:   "Incorrect request. POST only.",
		}
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")

	var jsonData []byte
	if statusCode != http.StatusNoContent {
		w.Header().Set("Content-Type", "application/json")
		jsonData, _ = json.Marshal(response)
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.WriteHeader(statusCode)
	if _, err := w.Write(jsonData); err != nil {
		logging.Error("API", "Failed to write pinfo response:", err)
	}
}

func handlePinfoImpl(r *http.Request) (PinfoResponse, int) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return PinfoResponse{
			Success: false,
			Error:   "Unable to read request body",
		}, http.StatusBadRequest
	}

	var req PinfoRequestSpec
	err = json.Unmarshal(body, &req)
	if err != nil {
		return PinfoResponse{
			Success: false,
			Error:   err.Error(),
		}, http.StatusBadRequest
	}

	if req.ProfileID == 0 {
		return PinfoResponse{
			Success: false,
			Error:   "Profile ID missing or 0 in request",
		}, http.StatusBadRequest
	}

	realUser, ok := db.GetProfile(req.ProfileID)
	if !ok {
		return PinfoResponse{
			Success: false,
			Error:   "Failed to find user in the database",
		}, http.StatusInternalServerError
	}

	fullAccess := apiSecret == "" || req.Secret == apiSecret

	user := realUser
	if !fullAccess {
		// Invalid or missing secret: return only the public-safe subset.
		user = database.User{
			ProfileId:  realUser.ProfileId,
			Restricted: realUser.Restricted,
			OpenHost:   realUser.OpenHost,
			DiscordID:  realUser.DiscordID,
		}
	}

	miiName := ""
	miiData := ""
	if fullAccess {
		miiName, miiData = getPinfoMiiData(realUser.ProfileId)
	}

	return PinfoResponse{
		Player: PinfoPlayer{
			ProfileID: user.ProfileId,
			MiiName:   miiName,
			MiiData:   miiData,
			OpenHost:  user.OpenHost,
			Banned:    user.Restricted,
			DiscordID: user.DiscordID,
		},
		Success: true,
		Error:   "",
	}, http.StatusOK
}

func getPinfoMiiData(profileID uint32) (string, string) {
	friendInfo := db.GetMKWFriendInfo(profileID)
	if friendInfo == "" {
		return "", ""
	}

	binaryData, err := base64.StdEncoding.DecodeString(friendInfo)
	if err != nil || len(binaryData) < 0x4C {
		return "", ""
	}

	mii := common.RawMiiFromBytes(binaryData)
	if mii.CalculateMiiCRC() != 0 {
		return "", ""
	}

	mii = mii.ClearMiiInfo()
	miiName, err := common.GetWideString(mii.Data[0x2:0x16], binary.BigEndian)
	if err != nil {
		miiName = ""
	}

	return miiName, base64.StdEncoding.EncodeToString(mii.Data[:])
}
