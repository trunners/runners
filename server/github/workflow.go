package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type Github struct {
	ID         string
	Owner      string
	Repository string
	Ref        string
	RunsOn     string
	Token      string
	URL        string
}

func New(token string) (Github, error) {
	if token == "" {
		return Github{}, errors.New("missing GitHub token")
	}

	return Github{
		Token: token,
	}, nil
}

type Inputs struct {
	RunsOn string `json:"runs-on"`
	Server string `json:"server"`
}

type Dispatch struct {
	Ref    string `json:"ref"`
	Inputs Inputs `json:"inputs"`
}

func (g Github) Workflow(ctx context.Context, id, owner, repository, ref, runsOn, server string) error {
	inputs := Inputs{
		RunsOn: runsOn,
		Server: server,
	}

	dispatch := Dispatch{
		Ref:    ref,
		Inputs: inputs,
	}

	body, err := json.Marshal(dispatch)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/workflows/%s/dispatches", owner, repository, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Accpet", "application/vnd.github+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.Token))
	req.Header.Set("X-Github-Api-Version", "2022-11-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to trigger workflow: %s", resp.Status)
	}

	return nil
}
