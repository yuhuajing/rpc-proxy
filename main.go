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

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
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
		if localchainhttpurl == "" || localchainwsurl == "" {
			cfg.URL = "http://127.0.0.1:8545"
			cfg.WSURL = "ws://127.0.0.1:8546"
			//log.Fatal("Need to specify a local Ethereum network")
		} else {
			cfg.URL = localchainhttpurl
			cfg.WSURL = localchainwsurl
		}

		if allowedscdeployer != "" {
			SCDeployers := strings.Split(allowedscdeployer, ",")
			for _, addr := range SCDeployers {
				SCAddress[strings.ToLower(addr)] = true
			}
		}

		if opendChainFunc != "" {
			allowdCMDS := strings.Split(opendChainFunc, ",")
			cfg.Allow = allowdCMDS
		} else {
			cfg.Allow = strings.Split("eth_blockNumber,eth_call,eth_chainId,eth_estimateGas,eth_gasPrice,eth_getBalance,eth_getBlockByHash,eth_getBlockByNumber,eth_getBlockTransactionCountByHash,eth_getBlockTransactionCountByNumber,eth_getCode,eth_getTransactionByBlockHashAndIndex,eth_getTransactionByBlockNumberAndIndex,eth_getTransactionByHash,eth_getTransactionCount,eth_getTransactionReceipt,eth_sendRawTransaction,net_listening,net_version", ",")
		}

		if portenv == "" {
			cfg.Port = 3000
		} else {
			port, _ := strconv.Atoi(portenv)
			cfg.Port = uint64(port)
		}

		if RPM == "" {
			requestsPerMinuteLimit = 1000
		} else {
			requestsPerMinuteLimit, _ = strconv.Atoi(RPM)
		}

		if ChainIDenv == "" {
			cfg.ChainID = 515
		} else {
			chainid, _ := strconv.Atoi(ChainIDenv)
			cfg.ChainID = int64(chainid)
		}

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
