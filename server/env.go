package main

import (
	"context"
	"net"
	"os"
)

type Env struct {
	Hostname           string
	WorkflowPort       string
	DevicePort         string
	WorkflowID         string
	WorkflowOwner      string
	WorkflowRepository string
	GithubToken        string
}

func LoadEnv(ctx context.Context) *Env {
	env := &Env{
		Hostname:           getEnv("HOSTNAME", getOutboundIP(ctx).String()),
		WorkflowPort:       getEnv("WORKFLOW_PORT", "8081"),
		DevicePort:         getEnv("DEVICE_PORT", "8080"),
		WorkflowID:         getEnv("WORKFLOW_ID", ""),
		WorkflowOwner:      getEnv("WORKFLOW_OWNER", ""),
		WorkflowRepository: getEnv("WORKFLOW_REPOSITORY", ""),
		GithubToken:        getEnv("GITHUB_TOKEN", ""),
	}

	return env
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		if defaultValue == "" {
			panic("Environment variable " + key + " is required but not set")
		}

		return defaultValue
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
