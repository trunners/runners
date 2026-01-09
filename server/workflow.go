package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type Workflow struct {
	ID         string
	Owner      string
	Repository string
	Token      string
	URL        string
}

func NewWorkflow(id, owner, repository, token string) (*Workflow, error) {
	if id == "" || owner == "" || repository == "" || token == "" {
		return nil, errors.New("missing required workflow parameters")
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/workflows/%s/dispatches", owner, repository, id)
	return &Workflow{
		ID:         id,
		Owner:      owner,
		Repository: repository,
		Token:      token,
		URL:        url,
	}, nil
}

type Inputs struct {
	Image string `json:"image"`
}

type Dispatch struct {
	Ref    string `json:"ref"`
	Inputs Inputs `json:"inputs"`
}

func (w Workflow) start(ctx context.Context, image string, ref string) error {
	inputs := Inputs{
		Image: image,
	}

	dispatch := Dispatch{
		Ref:    ref,
		Inputs: inputs,
	}

	body, err := json.Marshal(dispatch)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Accpet", "application/vnd.github+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", w.Token))
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
