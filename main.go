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
	app.Version = Version

	// err := godotenv.Load()
	// if err != nil {
	// 	log.Fatal("Error loading .env file")
	// }

	opendChainFunc := os.Getenv("ALLOW_CMDS")
	fmt.Println(opendChainFunc)
	allowedscdeployer := os.Getenv("ALLOW_CONTRACTS_DEPLOYER")
	fmt.Println(allowedscdeployer)
	portenv := os.Getenv("EXPORT_PORT")
	fmt.Println(portenv)
	localchainhttpurl := os.Getenv("ETHEREUM_HTTP_URL")
	fmt.Println(localchainhttpurl)
	localchainwsurl := os.Getenv("ETHETEUM_WS_URL")
	fmt.Println(localchainwsurl)
	ChainIDenv := os.Getenv("CHAIN_ID")
	fmt.Println(ChainIDenv)

	app.Action = func(c *cli.Context) error {
		var cfg ConfigData

		if allowedscdeployer != "" {
			SCDeployers := strings.Split(allowedscdeployer, ",")
			for _, addr := range SCDeployers {
				fmt.Println(addr)

				SCAddress[strings.ToLower(addr)] = true
			}
		}

		if opendChainFunc != "" {
			allowdCMDS := strings.Split(opendChainFunc, ",")
			cfg.Allow = allowdCMDS
			fmt.Println(allowdCMDS)
		}

		port, _ := strconv.Atoi(portenv)
		cfg.Port = uint64(port)
		if localchainhttpurl == "" || localchainwsurl == "" {
			log.Fatal("Need to specify a local Ethereum network")
		}
		cfg.URL = localchainhttpurl
		cfg.WSURL = localchainwsurl
		chainid, _ := strconv.Atoi(ChainIDenv)
		cfg.ChainID = int64(chainid)
		requestsPerMinuteLimit = 1000
		//cfg.BlockRangeLimit = 10

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

	gotils.L(ctx).Info().Println("Server starting, export port:", cfg.Port, "localchainhttpurl:", cfg.URL, "localchainwsurl:", cfg.WSURL,
		"rpmLimit:", cfg.RPM, "whitelistIP:", cfg.NoLimit, "opendChainFuncs:", cfg.Allow)

	// Create proxy server.
	server, err := cfg.NewServer()
	if err != nil {
		return fmt.Errorf("failed to start server: %s", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	// Use default options
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
	return http.ListenAndServe(":"+fmt.Sprint(cfg.Port), r)
}
