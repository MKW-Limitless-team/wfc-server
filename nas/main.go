package nas

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"wwfc/api"
	"wwfc/common"
	"wwfc/gamestats"
	"wwfc/logging"
	"wwfc/race"
	"wwfc/sake"

	"github.com/logrusorgru/aurora/v3"
)

var (
	serverName           string
	server, tlsServer    *http.Server
	payloadServerAddress string
)

var (
	authMux      = http.NewServeMux()
	dlsMux       = http.NewServeMux()
	sakeMux      = http.NewServeMux()
	gamestatsMux = http.NewServeMux()
	raceMux      = http.NewServeMux()
)

var hostMuxes = map[*regexp.Regexp]*http.ServeMux{
	regexp.MustCompile(`^(nas|naswii)\.`):                         authMux,
	regexp.MustCompile(`^dls1\.`):                                 dlsMux,
	regexp.MustCompile(`(\.|^)gamestats2?\.(gs\.|gamespy\.com$)`): gamestatsMux,
	regexp.MustCompile(`(\.|^)sake\.(gs\.|gamespy\.com$)`):        sakeMux,
	regexp.MustCompile(`(\.|^)race\.(gs\.|gamespy\.com$)`):        raceMux,
}

func StartServer(reload bool) {
	// Get config
	config := common.GetConfig()
	serverName = config.ServerName
	address := *config.NASAddress + ":" + config.NASPort
	payloadServerAddress = config.PayloadServerAddress

	if config.EnableHTTPS {
		go setupTLS(config)
	}

	err := CacheProfanityFile()
	if err != nil {
		logging.Info("NAS", err)
	}

	server = &http.Server{
		Addr:        address,
		Handler:     http.HandlerFunc(handleRequest),
		IdleTimeout: 20 * time.Second,
		ReadTimeout: 10 * time.Second,
	}

	authMux.HandleFunc("/ac", handleAuthAccountEndpoint)
	authMux.HandleFunc("/pr", handleAuthProfanityEndpoint)

	dlsMux.HandleFunc("/download", handleDownloadEndpoint)

	if payloadServerAddress != "" {
		// Forward the request to the payload server
		authMux.HandleFunc("/payload", forwardPayloadRequest)
		authMux.HandleFunc("/payload/", forwardPayloadRequest)
	} else {
		authMux.HandleFunc("/payload", handlePayloadRequest)
	}

	for i := 0; i <= 9; i++ {
		authMux.HandleFunc("/w"+strconv.Itoa(i), downloadStage1)
	}

	authMux.HandleFunc("/nastest.jsp", handleNASTest)

	http.HandleFunc("GET conntest.nintendowifi.net/", handleConnectionTest)

	api.RegisterHandlers(http.DefaultServeMux)
	sake.RegisterHandlers(sakeMux)
	race.RegisterHandlers(raceMux)
	gamestatsMux.HandleFunc("/", gamestats.HandleWebRequest)

	http.HandleFunc("/", handleUnknown)
	authMux.HandleFunc("/", handleUnknown)
	sakeMux.HandleFunc("/", handleUnknown)
	raceMux.HandleFunc("/", handleUnknown)

	go listenAndServe()
	if config.EnableHTTPS {
		tlsServer = &http.Server{
			Addr:        *config.NASAddressHTTPS + ":" + config.NASPortHTTPS,
			Handler:     server.Handler,
			IdleTimeout: server.IdleTimeout,
			ReadTimeout: server.ReadTimeout,
		}

		go func() {
			setupTLS(config)
			listenAndServeTLS()
		}()
	}
}

func Shutdown() {
	if server == nil {
		return
	}

	ctx, release := context.WithTimeout(context.Background(), 10*time.Second)
	defer release()

	err := server.Shutdown(ctx)
	if err != nil {
		logging.Error("NAS", "Error on HTTP shutdown:", err)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if blocked, rule := common.IsIPBanned(r.RemoteAddr); blocked {
		logging.Warn("NAS", "Blocked HTTP request from", aurora.BrightCyan(r.RemoteAddr), "matching", aurora.Cyan(rule))
		replyHTTPError(w, http.StatusForbidden, "403 Forbidden")
		return
	}

	// Check for *.sake.gs.* or sake.gs.*
	if regexSakeHost.MatchString(r.Host) {
		// Redirect to the sake server
		sake.HandleRequest(w, r)
		return
	}

	// Check for *.gamestats(2).gs.* or gamestats(2).gs.*
	if regexGamestatsHost.MatchString(r.Host) {
		// Redirect to the gamestats server
		gamestats.HandleWebRequest(w, r)
		return
	}

	// Check for *.race.gs.* or race.gs.*
	if regexRaceHost.MatchString(r.Host) {
		// Redirect to the race server
		race.HandleRequest(w, r)
		return
	}

	moduleName := "NAS:" + r.RemoteAddr

	// Handle conntest server
	if strings.HasPrefix(r.Host, "conntest.") {
		handleConnectionTest(w)
		return
	}

	// Handle DWC auth requests
	if r.URL.String() == "/ac" || r.URL.String() == "/pr" || r.URL.String() == "/download" {
		handleAuthRequest(moduleName, w, r)
		return
	}

	// Handle /nastest.jsp
	if r.URL.Path == "/nastest.jsp" {
		handleNASTest(w)
		return
	}

	// Check for /payload
	if strings.HasPrefix(r.URL.String(), "/payload") {
		logging.Info("NAS", aurora.Yellow(r.Method), aurora.Cyan(r.URL), "via", aurora.Cyan(r.Host), "from", aurora.BrightCyan(r.RemoteAddr))
		if payloadServerAddress != "" {
			// Forward the request to the payload server
			forwardPayloadRequest(moduleName, w, r)
		} else {
			handlePayloadRequest(moduleName, w, r)
		}
		return
	}

	// Stage 1
	if match := regexStage1URL.FindStringSubmatch(r.URL.String()); match != nil {
		val, err := strconv.Atoi(match[1])
		if err != nil {
			panic(err)
		}

		logging.Info("NAS", "Get stage 1:", aurora.Yellow(r.Method), aurora.Cyan(r.URL), "via", aurora.Cyan(r.Host), "from", aurora.BrightCyan(r.RemoteAddr))
		downloadStage1(w, val)
		return
	}

	// Check for /api/groups
	if r.URL.Path == "/api/groups" {
		api.HandleGroups(w, r)
		return
	}

	// Check for /api/pinfo
	if r.URL.Path == "/api/pinfo" {
		api.HandlePinfo(w, r)
		return
	}

	// Check for /api/stats
	if r.URL.Path == "/api/stats" {
		api.HandleStats(w, r)
		return
	}

	// Check for /api/ban
	if r.URL.Path == "/api/ban" {
		api.HandleBan(w, r)
		return
	}

	// Check for /api/unban
	if r.URL.Path == "/api/unban" {
		api.HandleUnban(w, r)
		return
	}

	// Check for /api/kick
	if r.URL.Path == "/api/kick" {
		api.HandleKick(w, r)
		return
	}

	// Check for /api/mkw_rr
	if r.URL.Path == "/api/mkw_rr" {
		api.HandleMKWRR(w, r)
		return
	}

	// Check for /api/mkw_tracks
	if r.URL.Path == "/api/mkw_tracks" {
		api.HandleMKWTracks(w, r)
		return
	}

	// Check for /api/mkw_characters
	if r.URL.Path == "/api/mkw_characters" {
		api.HandleMKWCharacters(w, r)
		return
	}

	// Check for /api/mkw_vehicles
	if r.URL.Path == "/api/mkw_vehicles" {
		api.HandleMKWVehicles(w, r)
		return
	}

	logging.Info("NAS", aurora.Yellow(r.Method), aurora.Cyan(r.URL), "via", aurora.Cyan(r.Host), "from", aurora.BrightCyan(r.RemoteAddr))
	replyHTTPError(w, 404, "404 Not Found")
}

func replyHTTPError(w http.ResponseWriter, errorCode int, errorString string) {
	response := "<html>\n" +
		"<head><title>" + errorString + "</title></head>\n" +
		"<body>\n" +
		"<center><h1>" + errorString + "</h1></center>\n" +
		"<hr><center>" + serverName + "</center>\n" +
		"</body>\n" +
		"</html>\n"

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	w.Header().Set("Connection", "close")
	w.Header().Set("Server", "Nintendo")
	w.WriteHeader(errorCode)
	_, _ = w.Write([]byte(response))
}

func handleUnknown(w http.ResponseWriter, r *http.Request) {
	logging.Info(getModuleName(r), "Unknown request:", aurora.Yellow(r.Method), aurora.Cyan(r.Host+r.URL.Path))
	replyHTTPError(w, http.StatusNotFound, "404 Not Found")
}

func handleNASTest(w http.ResponseWriter, r *http.Request) {
	response := "" +
		"<html>\n" +
		"<body>\n" +
		"</br>AuthServer is up</br> \n" +
		"\n" +
		"</body>\n" +
		"</html>\n"

	w.Header().Set("Content-Type", "text/html;charset=ISO-8859-1")
	w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	w.Header().Set("Connection", "close")
	w.Header().Set("NODE", "authserver-service.authserver.svc.cluster.local")
	w.Header().Set("Server", "Nintendo")

	w.WriteHeader(200)
	_, _ = w.Write([]byte(response))
}
