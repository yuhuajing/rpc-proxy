package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	toml "github.com/pelletier/go-toml"
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
	var configPath string
	var port uint64
	var localchainhttpurl string
	var localchainwsurl string
	var opendChainFunc string
	var noLimitIPs string
	var blockRangeLimit uint64

	app := cli.NewApp()
	app.Name = "rpc-proxy"
	app.Usage = "A proxy for web3 JSONRPC"
	app.Version = Version

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "config, c",
			Usage:       "path to toml config file",
			Destination: &configPath,
		},
		&cli.Uint64Flag{
			Name:        "port, p",
			Usage:       "port to export to",
			Destination: &port,
		},
		&cli.StringFlag{
			Name:        "url, u",
			Value:       "http://127.0.0.1:8545",
			Usage:       "local chain http url",
			Destination: &localchainhttpurl,
		},
		&cli.StringFlag{
			Name:        "wsurl, w",
			Value:       "ws://127.0.0.1:8546",
			Usage:       "local chain websocket url",
			Destination: &localchainwsurl,
		},
		&cli.StringFlag{
			Name:        "allowedfunc, a",
			Usage:       "comma separated list of allowed paths",
			Destination: &opendChainFunc,
		},
		&cli.IntFlag{
			Name:        "rpm",
			Usage:       "limit for number of requests per minute from single IP",
			Destination: &requestsPerMinuteLimit,
		},
		&cli.StringFlag{
			Name:        "nolimit, n",
			Usage:       "list of ips allowed unlimited requests(separated by commas)",
			Destination: &noLimitIPs,
		},
		&cli.Uint64Flag{
			Name:        "blocklimit, b",
			Usage:       "block range query limit",
			Destination: &blockRangeLimit,
		},
		&cli.Int64Flag{
			Name:        "chainId, id",
			Usage:       "chainId",
			Destination: &ChainID,
		},
	}

	app.Action = func(c *cli.Context) error {
		var cfg ConfigData
		if configPath != "" {
			t, err := toml.LoadFile(configPath)
			if err != nil {
				return err
			}
			if err := t.Unmarshal(&cfg); err != nil {
				return err
			}
			for _, addr := range cfg.SCAddress {
				SCAddress[strings.ToLower(addr)] = true
			}
		} else {
			return errors.New("CONFIG_TOML_NEEDED")
		}

		if port == 0 {
			port = cfg.Port
		}

		if localchainhttpurl != "" {
			cfg.URL = localchainhttpurl
		}
		if localchainwsurl != "" {
			cfg.WSURL = localchainwsurl
		}
		if requestsPerMinuteLimit == 0 {
			requestsPerMinuteLimit = cfg.RPM
		}
		if opendChainFunc != "" {
			cfg.Allow = strings.Split(opendChainFunc, ",")
		}
		if noLimitIPs != "" {
			cfg.NoLimit = strings.Split(noLimitIPs, ",")
		}
		if blockRangeLimit > 0 {
			cfg.BlockRangeLimit = blockRangeLimit
		}
		if ChainID == 0 {
			ChainID = cfg.ChainID
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
