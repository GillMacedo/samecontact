// Command samecontact finds duplicate people in contact and customer CSV files.
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/GillMacedo/samecontact/internal/contacts"
	"github.com/GillMacedo/samecontact/internal/server"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const version = "0.1.0"

const usage = `Usage:
  samecontact find <contacts.csv>   Find rows that are probably the same person
  samecontact serve                 Run as an MCP server over stdio
  samecontact version

Options for find:
  --columns email=Mail,name=Who     Name columns the tool doesn't recognize
  --out marked.csv                  Write a copy with dup_group and dup_level columns
  --json                            Print JSON

Reads your file and never changes it. Never uses the network.
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, usage)
		return 2
	}
	switch args[0] {
	case "serve":
		if err := server.New(version).Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			fmt.Fprintf(stderr, "samecontact: %v\n", err)
			return 2
		}
		return 0
	case "version":
		fmt.Fprintln(stdout, version)
		return 0
	case "find":
		if err := find(args[1:], stdout); err != nil {
			fmt.Fprintf(stderr, "samecontact: %v\n", err)
			return 2
		}
		return 0
	}
	fmt.Fprint(stdout, usage)
	return 2
}

func find(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("find", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	columnsFlag := flags.String("columns", "", "")
	out := flags.String("out", "", "")
	asJSON := flags.Bool("json", false, "")

	// The standard flag package stops at the first path, so options can't
	// follow it without this loop.
	var paths []string
	for rest := args; ; {
		if err := flags.Parse(rest); err != nil {
			return err
		}
		if flags.NArg() == 0 {
			break
		}
		paths = append(paths, flags.Arg(0))
		rest = flags.Args()[1:]
	}
	if len(paths) != 1 {
		return fmt.Errorf("find takes one CSV file\n%s", usage)
	}

	overrides := map[string]string{}
	if *columnsFlag != "" {
		for _, pair := range strings.Split(*columnsFlag, ",") {
			field, name, ok := strings.Cut(pair, "=")
			if !ok {
				return fmt.Errorf("--columns needs field=Column pairs, not %q", pair)
			}
			overrides[strings.TrimSpace(field)] = strings.TrimSpace(name)
		}
	}

	table, err := contacts.Load(paths[0], overrides)
	if err != nil {
		return err
	}
	result := contacts.Find(table.Contacts)

	if *out != "" {
		if err := writeMarked(paths[0], *out, table, result); err != nil {
			return err
		}
	}
	if *asJSON {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(struct {
			contacts.Result
			ExtraRows   int   `json:"extra_rows"`
			SkippedRows []int `json:"skipped_rows"`
		}{result, result.ExtraRows(), table.SkippedRow})
	}
	printText(stdout, table, result)
	return nil
}

func printText(w io.Writer, table *contacts.Table, result contacts.Result) {
	section := func(title, level string) {
		var lines []string
		for _, group := range result.Groups {
			if group.Level != level {
				continue
			}
			rows := make([]string, len(group.Rows))
			for i, row := range group.Rows {
				rows[i] = fmt.Sprint(row)
			}
			lines = append(lines, fmt.Sprintf("  rows %s  %s  [%s]", strings.Join(rows, ", "), strings.Join(group.Names, " / "), strings.Join(group.Reasons, "; ")))
		}
		if len(lines) > 0 {
			fmt.Fprintf(w, "%s (%d)\n%s\n", title, len(lines), strings.Join(lines, "\n"))
		}
	}
	section("SAME PERSON", contacts.LevelSame)
	section("LIKELY: check these", contacts.LevelLikely)
	if len(result.Review) > 0 {
		fmt.Fprintf(w, "REVIEW: a person should decide (%d)\n", len(result.Review))
		for _, pair := range result.Review {
			fmt.Fprintf(w, "  rows %d, %d  %s / %s  [%s]\n", pair.Rows[0], pair.Rows[1], pair.Names[0], pair.Names[1], pair.Reason)
		}
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(w, "WARNING: %s\n", warning)
	}
	for _, row := range table.SkippedRow {
		fmt.Fprintf(w, "SKIPPED row %d: no name, email or phone\n", row)
	}
	fmt.Fprintf(w, "%d contacts, %d duplicate groups, %d extra rows, %d pairs to review\n", result.Contacts, len(result.Groups), result.ExtraRows(), len(result.Review))
}

// writeMarked never overwrites anything, so it can't damage the input.
func writeMarked(input, output string, table *contacts.Table, result contacts.Result) error {
	in, err1 := filepath.Abs(input)
	target, err2 := filepath.Abs(output)
	if err1 != nil || err2 != nil || in == target {
		return fmt.Errorf("--out must be a different file from the input")
	}
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("can't create %s: %w. It must not already exist", output, err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	if err := writer.Write(append(append([]string{}, table.Header...), "dup_group", "dup_level")); err != nil {
		return err
	}
	labels := result.RowLabels()
	for offset, row := range table.Rows {
		label := labels[table.HeaderRow+offset+1]
		cells := append([]string{}, row...)
		for len(cells) < len(table.Header) {
			cells = append(cells, "")
		}
		if err := writer.Write(append(cells, label[0], label[1])); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
