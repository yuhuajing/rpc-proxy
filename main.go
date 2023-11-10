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
	opendChainFunc := os.Getenv("ALLOW_CMDS")
	allowedscdeployer := os.Getenv("ALLOW_CONTRACTS_DEPLOYER")
	portenv := os.Getenv("EXPORT_PORT")
	localchainhttpurl := os.Getenv("ETHEREUM_HTTP_URL")
	localchainwsurl := os.Getenv("ETHETEUM_WS_URL")
	ChainIDenv := os.Getenv("CHAIN_ID")
	RPM := os.Getenv("RPM_SERVER")

	//app.Action = func(c *cli.Context) error {
	var cfg ConfigData
	if localchainhttpurl == "" || localchainwsurl == "" {
		log.Fatal("Need to specify a local Ethereum network")
	}

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

	if RPM == "" {
		requestsPerMinuteLimit = 1000
	} else {
		requestsPerMinuteLimit, _ = strconv.Atoi(RPM)
	}
	cfg.URL = localchainhttpurl
	cfg.WSURL = localchainwsurl
	chainid, _ := strconv.Atoi(ChainIDenv)
	cfg.ChainID = int64(chainid)
	sort.Strings(cfg.Allow)
	sort.Strings(cfg.NoLimit)

	// Create proxy server.
	server, err := cfg.NewServer()
	if err != nil {
		fmt.Println(fmt.Errorf("failed to start server: %s", err))
		return
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
	r.HandleFunc("/zc", hello)
	http.ListenAndServe("0.0.0.0:3000", r)
	// if err := http.ListenAndServe("0.0.0.0:3000", r); err != nil {
	// 	panic(err)
	// }
	gotils.L(ctx).Info().Println("Server starting, export port:", cfg.Port, "localchainhttpurl:", cfg.URL, "localchainwsurl:", cfg.WSURL,
		"rpmLimit:", cfg.RPM, "whitelistIP:", cfg.NoLimit, "opendChainFuncs:", cfg.Allow)
}

func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello Docker Form Golang!")
}
