package main

// Low-level CDP example: use the cdp package directly to send raw protocol messages. d

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/masfu/chrome-go/cdp"
)

func main() {
	ctx := context.Background()

	conn := cdp.NewConnection("ws://localhost:9222/json/version")
	if err := conn.Connect(ctx); err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close()

	// List all targets.
	reader, err := conn.SendMessage(ctx, cdp.Message{
		Method: "Target.getTargets",
	})
	if err != nil {
		log.Fatalf("send message: %v", err)
	}

	resp, err := reader.Read()
	if err != nil {
		log.Fatalf("read response: %v", err)
	}

	var result struct {
		TargetInfos []struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
			URL      string `json:"url"`
			Title    string `json:"title"`
		} `json:"targetInfos"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		log.Fatalf("unmarshal: %v", err)
	}

	fmt.Printf("Found %d target(s):\n", len(result.TargetInfos))
	for _, t := range result.TargetInfos {
		fmt.Printf("  [%s] %s — %s\n", t.Type, t.Title, t.URL)
	}
}
