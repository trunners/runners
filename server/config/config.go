package config

import (
	"encoding/json"
	"os"
	"strconv"
)

type Workflow struct {
	Hostname   string `json:"hostname"`
	ID         string `json:"id"`
	Owner      string `json:"owner"`
	Repository string `json:"repo"`
	Ref        string `json:"ref"`
	RunsOn     string `json:"runs_on"`
}

type Config struct {
	Port        int        `json:"port"`
	GithubToken string     `json:"github_token"`
	Workflows   []Workflow `json:"workflows"`
}

func Load() (*Config, error) {
	location := env("CONFIG", "config.json")
	file, err := os.ReadFile(location)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = json.Unmarshal(file, &cfg)
	if err != nil {
		return nil, err
	}

	cfg.Port, err = strconv.Atoi(env("PORT", "8080"))
	if err != nil {
		return nil, err
	}

	cfg.GithubToken = env("GITHUB_TOKEN", "")

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
