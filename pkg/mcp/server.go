package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/euroopencost/euroopencost/internal/parser"
	"github.com/euroopencost/euroopencost/internal/pricing"
	"github.com/euroopencost/euroopencost/internal/pricing/hetzner"
	"github.com/euroopencost/euroopencost/internal/pricing/ionos"
	"github.com/euroopencost/euroopencost/internal/pricing/stackit"
	"github.com/euroopencost/euroopencost/internal/scoring"
)

// Server implements the Model Context Protocol (MCP) server.
type Server struct {
	Name    string
	Version string
}

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

// Error represents a JSON-RPC 2.0 error.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Start launches the MCP server on stdin/stdout.
func (s *Server) Start(ctx context.Context) error {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var req Request
			if err := decoder.Decode(&req); err != nil {
				if err == io.EOF {
					return nil
				}
				log.Printf("Error decoding request: %v", err)
				continue
			}

			resp := s.handleRequest(req)
			if err := encoder.Encode(resp); err != nil {
				log.Printf("Error encoding response: %v", err)
			}
		}
	}
}

func (s *Server) handleRequest(req Request) Response {
	switch req.Method {
	case "initialize":
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{},
				"serverInfo": map[string]string{
					"name":    s.Name,
					"version": s.Version,
				},
			},
		}
	case "tools/list":
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"tools": []map[string]any{
					{
						"name":        "get_cloud_costs",
						"description": "Calculates costs from a Terraform Plan JSON",
						"inputSchema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"plan_json": map[string]any{
									"type":        "string",
									"description": "The full JSON content of a terraform show -json plan.tfplan command",
								},
							},
							"required": []string{"plan_json"},
						},
					},
					{
						"name":        "get_sovereignty_score",
						"description": "Calculates the Sovereign Score for a given infrastructure",
						"inputSchema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"plan_json": map[string]any{
									"type":        "string",
									"description": "Terraform Plan JSON content",
								},
							},
							"required": []string{"plan_json"},
						},
					},
				},
			},
		}
	case "tools/call":
		var callParams struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &callParams); err != nil {
			return Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &Error{Code: -32602, Message: "Invalid tool call params"},
			}
		}

		switch callParams.Name {
		case "get_cloud_costs", "get_sovereignty_score":
			var args struct {
				PlanJSON string `json:"plan_json"`
			}
			if err := json.Unmarshal(callParams.Arguments, &args); err != nil {
				return Response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error:   &Error{Code: -32602, Message: "Invalid tool arguments"},
				}
			}

			// Parse
			p := parser.NewParser()
			resources, err := p.ParseReader(bytes.NewReader([]byte(args.PlanJSON)))
			if err != nil {
				return Response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error:   &Error{Code: -32001, Message: fmt.Sprintf("Parse error: %v", err)},
				}
			}

			// Calc
			calc := pricing.NewCalculator(pricing.NewClient(), hetzner.NewClient(), stackit.NewClient(), ionos.NewClient())
			resources, total, err := calc.Calculate(resources)
			if err != nil {
				return Response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error:   &Error{Code: -32002, Message: fmt.Sprintf("Pricing error: %v", err)},
				}
			}

			if callParams.Name == "get_cloud_costs" {
				return Response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: map[string]any{
						"resources": resources,
						"totals": map[string]float64{
							"hourly":  total.HourlyPrice,
							"monthly": total.MonthlyPrice,
						},
					},
				}
			} else {
				scoreInfo := scoring.CalculateSovereignScore(resources, total)
				return Response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result:  scoreInfo,
				}
			}

		default:
			return Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &Error{Code: -32601, Message: fmt.Sprintf("Tool %s not found", callParams.Name)},
			}
		}
	default:
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    -32601,
				Message: fmt.Sprintf("Method %s not found", req.Method),
			},
		}
	}
}
