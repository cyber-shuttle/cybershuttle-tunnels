package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/cyber-shuttle/cybershuttle-tunnels/server"
	"io"
	"net/http"
	"os"

	"github.com/fatedier/frp/client"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/util/log"
)

type ClientConfig struct {
	AgentID   string `json:"agent_id"`
	LocalIP   string `json:"local_ip"`
	LocalPort int    `json:"local_port"`
	Transport struct {
		BandwidthLimitMode string `json:"bandwidth_limit_mode"`
		Protocol           string `json:"protocol"`
	} `json:"transport"`
	Auth struct {
		Method string `json:"method"`
		Token  string `json:"token"`
	} `json:"auth"`
	ServerAddr string `json:"server_addr"`
	ServerPort int    `json:"server_port"`
	ServerAPI  string `json:"server_api"`
	Log        struct {
		Level string `json:"level"`
		To    string `json:"to"`
	} `json:"log"`
}

func RunClientInternal(clientConfig *ClientConfig) (error, int, chan error) {
	baseConfig := &v1.ProxyBaseConfig{}
	baseConfig.Name = clientConfig.AgentID
	baseConfig.Type = clientConfig.Transport.Protocol
	baseConfig.LocalIP = clientConfig.LocalIP
	baseConfig.LocalPort = clientConfig.LocalPort
	baseConfig.Transport.BandwidthLimitMode = clientConfig.Transport.BandwidthLimitMode
	proxyConfig := &v1.TCPProxyConfig{ProxyBaseConfig: *baseConfig}
	proxyConfig.RemotePort = clientConfig.ServerPort
	proxyConfigs := []v1.ProxyConfigurer{proxyConfig}

	commonConfig := &v1.ClientCommonConfig{}
	commonConfig.Auth.Method = v1.AuthMethod(clientConfig.Auth.Method)
	commonConfig.Auth.Token = clientConfig.Auth.Token
	commonConfig.ServerAddr = clientConfig.ServerAddr
	commonConfig.ServerPort = clientConfig.ServerPort
	commonConfig.Transport.Protocol = clientConfig.Transport.Protocol
	commonConfig.Log.Level = clientConfig.Log.Level
	commonConfig.Log.To = clientConfig.Log.To

	visitorCfgs := []v1.VisitorConfigurer{}

	portResp, err := getAvailableServerPort(clientConfig.ServerAPI, clientConfig.AgentID)
	if err != nil {
		log.Errorf("Error getting available server port: %v", err)
		return err, 0, nil
	}

	log.Infof("Available server port: %d", portResp.Port)
	proxyConfig.RemotePort = portResp.Port
	warning, err := validation.ValidateAllClientConfig(commonConfig, proxyConfigs, visitorCfgs)
	if warning != nil {
		fmt.Printf("WARNING: %v\n", warning)
	}
	if err != nil {
		return err, 0, nil
	}

	errCh := make(chan error)

	go func() {
		err := startService(commonConfig, proxyConfigs, visitorCfgs, clientConfig.ServerAPI)
		if err != nil {
			log.Errorf("Error starting service: %v", err)
			errCh <- err
		}
	}()

	return nil, portResp.Port, errCh
}

func RunClient(cfgFilePath string) (error, int, chan error) {

	f, err := os.Open(cfgFilePath)
	if err != nil {
		return err, 0, nil
	}
	defer f.Close()

	raw, err := io.ReadAll(f)
	if err != nil {
		return err, 0, nil
	}

	var cfg ClientConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return err, 0, nil
	}
	return RunClientInternal(&cfg)
}

func startService(
	cfg *v1.ClientCommonConfig,
	proxyCfgs []v1.ProxyConfigurer,
	visitorCfgs []v1.VisitorConfigurer,
	apiUrl string,
) error {
	log.InitLogger(cfg.Log.To, cfg.Log.Level, int(cfg.Log.MaxDays), cfg.Log.DisablePrintColor)

	tcp := v1.NewProxyConfigurerByType(v1.ProxyTypeTCP)
	if tcp == nil {
		return fmt.Errorf("failed to create TCP proxy configurer")
	}

	svr, err := client.NewService(client.ServiceOptions{
		Common:      cfg,
		ProxyCfgs:   proxyCfgs,
		VisitorCfgs: visitorCfgs,
	})
	if err != nil {
		return err
	}

	return svr.Run(context.Background())
}

func getAvailableServerPort(apiUrl string, agentId string) (server.ReservePortResponse, error) {

	requestData := server.ReservePortRequest{
		AgentID: agentId,
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		log.Errorf("Error marshaling request data: %v", err)
		return server.ReservePortResponse{}, err
	}

	log.Infof("Api url: %s", apiUrl)
	resp, err := http.Post(apiUrl+"/reserve_port", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Errorf("Error making POST request: %v", err)
		return server.ReservePortResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Errorf("Error response from server: %s", resp.Status)
		return server.ReservePortResponse{}, fmt.Errorf("error response from server: %s", resp.Status)
	}
	var response server.ReservePortResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Errorf("Error decoding response: %v", err)
		return server.ReservePortResponse{}, err
	}
	if !response.Success {
		log.Errorf("Error from server: %s", response.Message)
		return server.ReservePortResponse{}, fmt.Errorf("error from server: %s", response.Message)
	}
	return response, nil
}
