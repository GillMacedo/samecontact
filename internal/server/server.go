// Package server exposes the duplicate finder as MCP tools.
package server

import (
	"context"
	"errors"

	"github.com/GillMacedo/samecontact/internal/contacts"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// FindInput is the input of contacts_find_duplicates.
type FindInput struct {
	Path    string            `json:"path" jsonschema:"Path to the contact CSV file"`
	Columns map[string]string `json:"columns,omitempty" jsonschema:"Column names by field (email, phone, name, first, last, company, city), when detection fails"`
	Limit   *int              `json:"limit,omitempty" jsonschema:"Most groups and review pairs to return, default 100. 0 returns only the counts"`
}

// FindOutput is the output of contacts_find_duplicates.
type FindOutput struct {
	Contacts    int              `json:"contacts"`
	ExtraRows   int              `json:"extra_rows"`
	GroupCount  int              `json:"group_count"`
	ReviewCount int              `json:"review_count"`
	Truncated   bool             `json:"truncated"`
	Groups      []contacts.Group `json:"groups"`
	Review      []contacts.Pair  `json:"review"`
	Warnings    []string         `json:"warnings"`
	SkippedRows []int            `json:"skipped_rows"`
}

// InspectInput is the input of contacts_inspect_csv.
type InspectInput struct {
	Path string `json:"path" jsonschema:"Path to the contact CSV file"`
}

// InspectOutput is the output of contacts_inspect_csv. It holds no row contents.
type InspectOutput struct {
	Columns     map[string]string `json:"columns"`
	HeaderRow   int               `json:"header_row"`
	Contacts    int               `json:"contacts"`
	SkippedRows int               `json:"skipped_rows"`
}

func yes() *bool { value := true; return &value }
func no() *bool  { value := false; return &value }

// New returns the MCP server.
func New(version string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "samecontact", Version: version}, nil)
	annotations := &mcp.ToolAnnotations{ReadOnlyHint: true, DestructiveHint: no(), IdempotentHint: true, OpenWorldHint: no()}

	mcp.AddTool(server, &mcp.Tool{
		Name:  "contacts_find_duplicates",
		Title: "Find duplicate contacts",
		Description: "Finds rows in a contact or customer CSV that are probably the same person, using emails, phone numbers, names and companies. " +
			"Use to clean a CRM export, mailing list or customer list. Read-only, and never uses the network. " +
			"Results include names from the file, so they reach the model. Report review pairs to the user instead of deciding them.",
		Annotations: annotations,
	}, find)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "contacts_inspect_csv",
		Title:       "Inspect a contact CSV",
		Description: "Shows which columns samecontact detects in a CSV and how many rows it can read, without returning row contents. Use if the file is unfamiliar or a search found nothing.",
		Annotations: annotations,
	}, inspect)
	return server
}

func find(_ context.Context, _ *mcp.CallToolRequest, in FindInput) (*mcp.CallToolResult, FindOutput, error) {
	limit := 100
	if in.Limit != nil {
		limit = *in.Limit
	}
	if limit < 0 || limit > 500 {
		return nil, FindOutput{}, errors.New("limit must be between 0 and 500")
	}
	table, err := contacts.Load(in.Path, in.Columns)
	if err != nil {
		return nil, FindOutput{}, err
	}
	result := contacts.Find(table.Contacts)
	out := FindOutput{
		Contacts: result.Contacts, ExtraRows: result.ExtraRows(), GroupCount: len(result.Groups), ReviewCount: len(result.Review),
		Groups: result.Groups, Review: result.Review, Warnings: result.Warnings, SkippedRows: table.SkippedRow,
	}
	if out.SkippedRows == nil {
		out.SkippedRows = []int{}
	}
	if len(out.Groups) > limit {
		out.Groups, out.Truncated = out.Groups[:limit], true
	}
	if len(out.Review) > limit {
		out.Review, out.Truncated = out.Review[:limit], true
	}
	return nil, out, nil
}

func inspect(_ context.Context, _ *mcp.CallToolRequest, in InspectInput) (*mcp.CallToolResult, InspectOutput, error) {
	table, err := contacts.Load(in.Path, nil)
	if err != nil {
		return nil, InspectOutput{}, err
	}
	return nil, InspectOutput{Columns: table.Columns, HeaderRow: table.HeaderRow, Contacts: len(table.Contacts), SkippedRows: len(table.SkippedRow)}, nil
}
