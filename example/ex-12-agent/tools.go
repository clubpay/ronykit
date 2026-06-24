package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/clubpay/ronykit/intent"
)

func newToolRegistry() (*intent.DefaultToolRegistry, error) {
	reg := intent.NewToolRegistry()

	err := reg.Register(intent.LocalToolFunc{
		Def: intent.ToolDefinition{
			Name:        "get_time",
			Description: "Returns the current UTC time in RFC3339 format.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		Fn: func(_ context.Context, _ json.RawMessage) (intent.Message, error) {
			now := time.Now().UTC().Format(time.RFC3339)

			return intent.Message{
				Role:  intent.RoleTool,
				Parts: []intent.Part{intent.TextPart(now)},
			}, nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("register get_time: %w", err)
	}

	// issue_refund is gated behind the "billing" skill: it stays hidden from the
	// model until that skill is activated via the activate_skill tool.
	err = reg.Register(intent.LocalToolFunc{
		Def: intent.ToolDefinition{
			Name:        "issue_refund",
			Description: "Issues a refund for the given order ID.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"orderId": map[string]any{
						"type":        "string",
						"description": "The order to refund.",
					},
				},
				"required": []string{"orderId"},
			},
		},
		Fn: func(_ context.Context, args json.RawMessage) (intent.Message, error) {
			var req struct {
				OrderID string `json:"orderId"`
			}
			_ = json.Unmarshal(args, &req)

			return intent.Message{
				Role:  intent.RoleTool,
				Parts: []intent.Part{intent.TextPart(fmt.Sprintf("refund issued for order %q", req.OrderID))},
			}, nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("register issue_refund: %w", err)
	}

	return reg, nil
}
