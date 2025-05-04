package server

import (
	"context"
	"fmt"
	"os"

	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/server"
)

var (
	cfgFile          string
	showVersion      bool
	strictConfigMode bool

	serverCfg v1.ServerConfig
)

func RunServer(cfgFilePath string) error {
	var (
		svrCfg         *v1.ServerConfig
		isLegacyFormat bool
		err            error
	)

	cfgFile = cfgFilePath

	if cfgFile != "" {
		svrCfg, isLegacyFormat, err = config.LoadServerConfig(cfgFilePath, strictConfigMode)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
			return nil
		}
		if isLegacyFormat {
			fmt.Printf("WARNI``NG: ini format is deprecated and the support will be removed in the future, " +
				"please use yaml/json/toml format instead!\n")
		}
	} else {
		serverCfg.Complete()
		svrCfg = &serverCfg
	}

	warning, err := validation.ValidateServerConfig(svrCfg)
	if warning != nil {
		fmt.Printf("WARNING: %v\n", warning)
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return startServer(svrCfg)
}

func startServer(cfg *v1.ServerConfig) (err error) {
	log.InitLogger(cfg.Log.To, cfg.Log.Level, int(cfg.Log.MaxDays), cfg.Log.DisablePrintColor)

	svr, err := server.NewService(cfg)
	if err != nil {
		return err
	}
	log.Infof("frps started successfully")
	svr.Run(context.Background())
	return
}
