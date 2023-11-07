package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func reportToSlack(slackHook string, report Report) error {
	var message strings.Builder
	message.WriteString("Listing undeletable resources")
	if clusterType := os.Getenv("CLUSTER_TYPE"); clusterType != "" {
		message.WriteString(" for cluster " + clusterType)
	}
	message.WriteRune('\n')
	for _, resource := range report.FailedToDelete {
		message.WriteString(fmt.Sprintf("%s: %q\n", resource.Type(), resource.ID()))
	}

	var msg bytes.Buffer
	if err := json.NewEncoder(&msg).Encode(struct {
		Text string `json:"text"`
	}{
		Text: message.String(),
	}); err != nil {
		return fmt.Errorf("failed to build the JSON payload for Slack: %w", err)
	}

	res, err := http.Post(
		slackHook,
		"application/json",
		&msg,
	)
	if err != nil {
		return fmt.Errorf("failed to send a message to Slack: %w", err)
	}

	io.Copy(io.Discard, res.Body)
	res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK, http.StatusNoContent, http.StatusAccepted:
	default:
		return fmt.Errorf("unexpected status code %q while sending a Slack notification", res.Status)
	}

	return nil
}
