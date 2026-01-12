package config

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"strconv"

	"golang.org/x/crypto/ssh"
)

type Workflow struct {
	ID         string `json:"id"`
	Owner      string `json:"owner"`
	Repository string `json:"repo"`
	Ref        string `json:"ref"`
	RunsOn     string `json:"runs-on"`
}

type Config struct {
	GithubToken    string
	Host           string
	Port           int
	AuthorizedKeys []ssh.PublicKey
	Server         *ssh.ServerConfig
	Client         *ssh.ClientConfig
	Workflows      map[string]Workflow
}

func Load(ctx context.Context) (*Config, error) {
	// Load config file
	location := env("CONFIG", "config.json")
	file, err := os.ReadFile(location)
	if err != nil {
		return nil, err
	}

	// Parse config file
	var cfg Config
	err = json.Unmarshal(file, &cfg.Workflows)
	if err != nil {
		return nil, err
	}

	// Require GitHub token
	cfg.GithubToken = env("GITHUB_TOKEN", "")

	// Parse host & port
	cfg.Host = env("HOST", getOutboundIP(ctx).String())
	cfg.Port, err = strconv.Atoi(env("PORT", "8080"))
	if err != nil {
		return nil, err
	}

	// Load authorized keys
	authorizedKeysFile := env("AUTHORIZED_KEYS", "/etc/ssh/authorized_keys")
	authorizedKeysBytes, err := os.ReadFile(authorizedKeysFile)
	if err != nil {
		return nil, err
	}
	for len(authorizedKeysBytes) > 0 {
		var key ssh.PublicKey
		var rest []byte
		key, _, _, rest, err = ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			return nil, err
		}

		cfg.AuthorizedKeys = append(cfg.AuthorizedKeys, key)
		authorizedKeysBytes = rest
	}

	cfg.Server = serverConfig(cfg.AuthorizedKeys)
	cfg.Client = clientConfig()

	return &cfg, nil
}

func env(key string, deafult string) string {
	value := os.Getenv(key)
	if value == "" {
		if deafult == "" {
			panic("Environment variable " + key + " is required but not set")
		}

		return deafult
	}

	return value
}

// get preferred outbound ip of this machine.
func getOutboundIP(ctx context.Context) net.IP {
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "udp", "8.8.8.8:80")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr()
	udpAddr, ok := localAddr.(*net.UDPAddr)
	if !ok {
		return net.IPv4zero
	}

	return udpAddr.IP
}
