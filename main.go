package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"github.com/treeder/gcputils"
	"github.com/treeder/gotils/v2"
	"github.com/urfave/cli/v2"
)

type ConfigData struct {
	Port            uint64   `toml:",omitempty"` //监听的端口
	URL             string   `toml:",omitempty"` //redirect url
	WSURL           string   `toml:",omitempty"`
	Allow           []string `toml:",omitempty"`
	RPM             int      `toml:",omitempty"`
	NoLimit         []string `toml:",omitempty"`
	BlockRangeLimit uint64   `toml:",omitempty"`
	SCAddress       []string `toml:",omitempty"`
	ChainID         int64    `toml:",omitempty"`
}

var SCAddress = make(map[string]bool)
var requestsPerMinuteLimit int
var ChainID int64

func main() {
	ctx := context.Background()
	gotils.SetLoggable(gcputils.NewLogger())

	app := cli.NewApp()
	app.Name = "rpc-proxy"
	app.Usage = "A proxy for web3 JSONRPC"

	var enverr error
	file, err := os.Open("/app/.env")
	if err != nil {
		file.Close()
		enverr = godotenv.Load()
	} else {
		file.Close()
		enverr = godotenv.Load("/app/.env")
	}

	if enverr != nil {
		log.Fatal("Error loading env file")
	}

	if enverr != nil {
		log.Fatal("Error loading env file")
	}

	opendChainFunc := os.Getenv("ALLOW_CMDS")
	allowedscdeployer := os.Getenv("ALLOW_CONTRACTS_DEPLOYER")
	portenv := os.Getenv("EXPORT_PORT")
	localchainhttpurl := os.Getenv("ETHEREUM_HTTP_URL")
	localchainwsurl := os.Getenv("ETHETEUM_WS_URL")
	ChainIDenv := os.Getenv("CHAIN_ID")
	RPM := os.Getenv("RPM_SERVER")

	app.Action = func(c *cli.Context) error {
		var cfg ConfigData

		if allowedscdeployer != "" {
			cfg.SCAddress = strings.Split(allowedscdeployer, ",")
			for _, addr := range cfg.SCAddress {
				SCAddress[strings.ToLower(addr)] = true
			}
		}

		cfg.URL = localchainhttpurl
		cfg.WSURL = localchainwsurl
		allowdCMDS := strings.Split(opendChainFunc, ",")
		cfg.Allow = allowdCMDS
		port, _ := strconv.Atoi(portenv)
		cfg.Port = uint64(port)
		requestsPerMinuteLimit, _ = strconv.Atoi(RPM)
		chainid, _ := strconv.Atoi(ChainIDenv)
		cfg.ChainID = int64(chainid)
		return cfg.run(ctx)
	}

	if err := app.Run(os.Args); err != nil {
		gotils.L(ctx).Error().Printf("Fatal error: %v", err)
		return
	}
	gotils.L(ctx).Info().Print("Shutting down")
}

func (cfg *ConfigData) run(ctx context.Context) error {
	sort.Strings(cfg.Allow)
	sort.Strings(cfg.NoLimit)

	gotils.L(ctx).Info().Println("Server starting, export port:", cfg.Port, "chainID", cfg.ChainID, "localchainhttpurl:", cfg.URL, "localchainwsurl:", cfg.WSURL,
		"rpmLimit:", requestsPerMinuteLimit, "SCdeployer:", cfg.SCAddress, "opendChainFuncs:", cfg.Allow)

	// Create proxy server.
	server, err := cfg.NewServer()
	if err != nil {
		return fmt.Errorf("failed to start server: %s", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"HEAD", "GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           3600,
	}).Handler)

	r.Head("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.HandleFunc("/*", server.RPCProxy)
	r.HandleFunc("/ws", server.WSProxy)
	return http.ListenAndServe("0.0.0.0:"+fmt.Sprint(cfg.Port), r)
}
