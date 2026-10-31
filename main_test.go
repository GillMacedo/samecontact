package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GillMacedo/samecontact/internal/server"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const sample = "examples/contacts.csv"

func TestFindPrintsGroups(t *testing.T) {
	var out, errs bytes.Buffer
	if code := run([]string{"find", sample}, &out, &errs); code != 0 {
		t.Fatalf("exit %d: %s", code, errs.String())
	}
	for _, want := range []string{"SAME PERSON (2)", "LIKELY: check these (1)", "REVIEW: a person should decide (2)", "SKIPPED row 16"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, out.String())
		}
	}
}

func TestOptionsCanFollowThePath(t *testing.T) {
	var out, errs bytes.Buffer
	if code := run([]string{"find", sample, "--json"}, &out, &errs); code != 0 {
		t.Fatalf("exit %d: %s", code, errs.String())
	}
	var body map[string]any
	if err := json.Unmarshal(out.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["extra_rows"] != float64(3) {
		t.Errorf("got %v", body["extra_rows"])
	}
}

func TestOutMarksRowsAndRefusesToOverwrite(t *testing.T) {
	target := filepath.Join(t.TempDir(), "marked.csv")
	var out, errs bytes.Buffer
	if code := run([]string{"find", sample, "--out", target}, &out, &errs); code != 0 {
		t.Fatalf("exit %d: %s", code, errs.String())
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.HasPrefix(text, "Name,Email,Phone,Company,City,dup_group,dup_level\n") || !strings.Contains(text, ",1,same\n") || !strings.Contains(text, ",,\n") {
		t.Errorf("got:\n%s", text)
	}
	if code := run([]string{"find", sample, "--out", target}, &out, &errs); code != 2 {
		t.Error("existing output should be refused")
	}
	before, _ := os.ReadFile(sample)
	if code := run([]string{"find", sample, "--out", sample}, &out, &errs); code != 2 {
		t.Error("overwriting the input should be refused")
	}
	after, _ := os.ReadFile(sample)
	if !bytes.Equal(before, after) {
		t.Error("input changed")
	}
}

func TestErrorsExitTwo(t *testing.T) {
	var out, errs bytes.Buffer
	for _, args := range [][]string{{"find"}, {"find", "/no/such.csv"}, {"nope"}, {}, {"find", sample, "--columns", "oops"}} {
		if code := run(args, &out, &errs); code != 2 {
			t.Errorf("%v: exit %d", args, code)
		}
	}
}

func connect(t *testing.T) *mcp.ClientSession {
	t.Helper()
	clientSide, serverSide := mcp.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := server.New("test").Connect(ctx, serverSide, nil); err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(ctx, clientSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func call(t *testing.T, name string, args map[string]any) (map[string]any, *mcp.CallToolResult) {
	t.Helper()
	result, err := connect(t).CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	body := map[string]any{}
	if !result.IsError {
		data, _ := json.Marshal(result.StructuredContent)
		json.Unmarshal(data, &body)
	}
	return body, result
}

func TestServerListsReadOnlyTools(t *testing.T) {
	list, err := connect(t).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Tools) != 2 {
		t.Fatalf("got %d tools", len(list.Tools))
	}
	for _, tool := range list.Tools {
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("%s isn't read-only", tool.Name)
		}
	}
}

func TestServerFinds(t *testing.T) {
	body, _ := call(t, "contacts_find_duplicates", map[string]any{"path": sample})
	if body["group_count"] != float64(3) || body["review_count"] != float64(2) {
		t.Errorf("got %v", body)
	}
	body, _ = call(t, "contacts_find_duplicates", map[string]any{"path": sample, "limit": 0})
	if body["truncated"] != true || len(body["groups"].([]any)) != 0 || body["group_count"] != float64(3) {
		t.Errorf("got %v", body)
	}
}

func TestServerInspectHasNoRowContents(t *testing.T) {
	body, result := call(t, "contacts_inspect_csv", map[string]any{"path": sample})
	if body["contacts"] != float64(13) {
		t.Errorf("got %v", body)
	}
	data, _ := json.Marshal(result)
	if strings.Contains(string(data), "Ana Silva") {
		t.Error("inspect returned row contents")
	}
}

func TestServerBadPathIsAToolError(t *testing.T) {
	_, result := call(t, "contacts_find_duplicates", map[string]any{"path": "/no/such.csv"})
	if !result.IsError {
		t.Error("want a tool error")
	}
}

func TestSourceHasNoNetworkCode(t *testing.T) {
	for _, file := range []string{"main.go", "internal/contacts/load.go", "internal/contacts/find.go", "internal/contacts/normalize.go", "internal/server/server.go"} {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, banned := range []string{`"net/http"`, `"net"`} {
			if strings.Contains(string(data), banned) {
				t.Errorf("%s imports %s", file, banned)
			}
		}
	}
}
