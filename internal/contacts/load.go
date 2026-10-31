// Package contacts finds likely duplicate people in a contact export.
package contacts

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxFileBytes = 50_000_000

// Contact is one row of the file, with row counted from 1 like a spreadsheet.
type Contact struct {
	Row     int
	Name    string
	Emails  []string
	Phones  []string
	Company string
	City    string
	cells   string
}

// Table is a loaded file.
type Table struct {
	Header     []string
	Rows       [][]string
	HeaderRow  int
	Columns    map[string]string
	Contacts   []Contact
	SkippedRow []int
}

var fieldNames = map[string][]string{
	"name":    {"name", "full name", "contact", "contact name", "customer", "customer name", "client", "client name", "lead name", "nome", "nome completo", "nombre", "cliente"},
	"first":   {"first name", "firstname", "given name", "first", "primeiro nome"},
	"last":    {"last name", "lastname", "surname", "family name", "last", "sobrenome", "apellido"},
	"company": {"company", "company name", "organization", "organisation", "account", "account name", "business", "empresa", "compania"},
	"city":    {"city", "town", "cidade", "ciudad"},
}

var emailColumn = regexp.MustCompile(`e-?mail|correo`)
var phoneColumn = regexp.MustCompile(`phone|mobile|cell|\btel\b|telephone|telefone|celular|whatsapp|telefono`)

func normalHeader(header string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(strings.ReplaceAll(plain(header), "_", " "), ".", " ")), " ")
}

type columns struct {
	single map[string]int
	emails []int
	phones []int
}

func detectColumns(header []string, overrides map[string]string) (columns, error) {
	found := columns{single: map[string]int{}}
	normal := make([]string, len(header))
	for i, cell := range header {
		normal[i] = normalHeader(cell)
	}
	for i, cell := range normal {
		switch {
		case emailColumn.MatchString(cell):
			found.emails = append(found.emails, i)
		case phoneColumn.MatchString(cell):
			found.phones = append(found.phones, i)
		}
		for field, names := range fieldNames {
			for _, name := range names {
				if _, set := found.single[field]; !set && cell == name {
					found.single[field] = i
				}
			}
		}
	}
	for field, name := range overrides {
		index := -1
		for i, cell := range normal {
			if cell == normalHeader(name) {
				index = i
			}
		}
		if index < 0 {
			return found, fmt.Errorf("no column named %q. Columns found: %s", name, strings.Join(header, ", "))
		}
		switch field {
		case "email":
			found.emails = []int{index}
		case "phone":
			found.phones = []int{index}
		case "name", "first", "last", "company", "city":
			found.single[field] = index
		default:
			return found, fmt.Errorf("unknown field %q. Use email, phone, name, first, last, company or city", field)
		}
	}
	return found, nil
}

func (c columns) score() int {
	score := len(c.single)
	if len(c.emails) > 0 {
		score += 2
	}
	if len(c.phones) > 0 {
		score += 2
	}
	return score
}

func decode(data []byte) string {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if utf8.Valid(data) {
		return string(data)
	}
	// Exports from older systems are often Latin-1, where each byte is a rune.
	runes := make([]rune, len(data))
	for i, b := range data {
		runes[i] = rune(b)
	}
	return string(runes)
}

func detectDelimiter(text string) rune {
	lines := strings.Split(text, "\n")
	if len(lines) > 30 {
		lines = lines[:30]
	}
	best, bestCount := ',', -1
	for _, delimiter := range []rune{',', ';', '\t', '|'} {
		count := 0
		for _, line := range lines {
			count += strings.Count(line, string(delimiter))
		}
		if count > bestCount {
			best, bestCount = delimiter, count
		}
	}
	return best
}

// Load reads a CSV contact export. Overrides name columns by field, such as
// email=Mail, for files whose headers aren't recognized.
func Load(path string, overrides map[string]string) (*Table, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("can't read %s: %w", path, errors.Unwrap(err))
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a folder. Pass a CSV file", path)
	}
	if info.Size() > maxFileBytes {
		return nil, fmt.Errorf("%s is larger than 50 MB", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("can't read %s: %w", path, err)
	}
	text := decode(data)
	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = detectDelimiter(text)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("can't parse %s as CSV: %w", path, err)
	}

	// Exports can have title lines above the header, so take the best of the
	// first 20 rows that names something we can match on.
	bestIndex, bestScore := -1, 0
	var cols columns
	for i := 0; i < len(rows) && i < 20; i++ {
		found, err := detectColumns(rows[i], overrides)
		if err != nil {
			continue
		}
		usable := len(found.emails) > 0 || len(found.phones) > 0 || hasName(found)
		if usable && found.score() > bestScore {
			bestIndex, bestScore, cols = i, found.score(), found
		}
	}
	if bestIndex < 0 {
		first := "(empty file)"
		if len(rows) > 0 {
			first = strings.Join(rows[0], ", ")
		}
		if _, err := detectColumns(firstRow(rows), overrides); err != nil && len(overrides) > 0 {
			return nil, err
		}
		return nil, fmt.Errorf("can't find email, phone or name columns in %s. First row: %s. Name them with a column map", path, first)
	}

	table := &Table{Header: rows[bestIndex], Rows: rows[bestIndex+1:], HeaderRow: bestIndex + 1, Columns: describe(rows[bestIndex], cols)}
	for offset, row := range table.Rows {
		contact := build(row, cols, bestIndex+offset+2)
		if contact.Name == "" && len(contact.Emails) == 0 && len(contact.Phones) == 0 {
			table.SkippedRow = append(table.SkippedRow, contact.Row)
			continue
		}
		table.Contacts = append(table.Contacts, contact)
	}
	return table, nil
}

func firstRow(rows [][]string) []string {
	if len(rows) == 0 {
		return nil
	}
	return rows[0]
}

func hasName(c columns) bool {
	_, name := c.single["name"]
	_, last := c.single["last"]
	return name || last
}

func describe(header []string, c columns) map[string]string {
	out := map[string]string{}
	for field, index := range c.single {
		out[field] = header[index]
	}
	var names []string
	for _, index := range c.emails {
		names = append(names, header[index])
	}
	if names != nil {
		out["email"] = strings.Join(names, ", ")
	}
	names = nil
	for _, index := range c.phones {
		names = append(names, header[index])
	}
	if names != nil {
		out["phone"] = strings.Join(names, ", ")
	}
	return out
}

func build(row []string, c columns, number int) Contact {
	cell := func(index int) string {
		if index < len(row) {
			return strings.TrimSpace(row[index])
		}
		return ""
	}
	field := func(name string) string {
		if index, ok := c.single[name]; ok {
			return cell(index)
		}
		return ""
	}
	contact := Contact{Row: number, Name: field("name"), Company: field("company"), City: field("city"), cells: strings.ToLower(strings.Join(row, "\x00"))}
	if contact.Name == "" {
		contact.Name = strings.TrimSpace(field("first") + " " + field("last"))
	}
	split := func(indexes []int) []string {
		var out []string
		for _, index := range indexes {
			for _, part := range strings.FieldsFunc(cell(index), func(r rune) bool { return r == ';' || r == '/' }) {
				if part = strings.TrimSpace(part); part != "" {
					out = append(out, part)
				}
			}
		}
		return out
	}
	contact.Emails = split(c.emails)
	contact.Phones = split(c.phones)
	return contact
}
