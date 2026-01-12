package config

import (
	"net"
	"os"
	"strconv"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	Port     int
	Address  string
	Shell    string
	Listener net.ListenConfig
	Dialer   *net.Dialer
	Server   *ssh.ServerConfig
}

func Load() *Config {
	var cfg Config
	var err error

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	cfg.Port, err = strconv.Atoi(port)
	if err != nil {
		panic(err)
	}

	cfg.Address = os.Getenv("SERVER_ADDRESS")
	if cfg.Address == "" {
		panic("SERVER_ADDRESS environment variable is required")
	}

	cfg.Shell = os.Getenv("SHELL")
	if cfg.Shell == "" {
		cfg.Shell = "bash"
	}

	cfg.Listener = net.ListenConfig{}
	cfg.Dialer = &net.Dialer{}
	cfg.Server = serverConfig()

	return &cfg
}
