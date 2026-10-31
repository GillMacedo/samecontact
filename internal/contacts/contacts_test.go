package contacts

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	cases := map[string]string{
		"Ana.Silva@Example.com ": "ana.silva@example.com",
		"mailto:a@example.com":   "a@example.com",
		"a+news@example.com":     "a@example.com",
		"a.b.c+x@gmail.com":      "abc@gmail.com",
		"a.b@googlemail.com":     "ab@gmail.com",
	}
	for raw, want := range cases {
		if got, ok := NormalizeEmail(raw); !ok || got != want {
			t.Errorf("%q: got %q", raw, got)
		}
	}
	for _, bad := range []string{"", "nope", "@example.com", "a@", "a@b", "a b@example.com", "a@example.com, b@example.com"} {
		if _, ok := NormalizeEmail(bad); ok {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestNormalizePhone(t *testing.T) {
	want := "5550100101"
	for _, raw := range []string{"+1 (555) 010-0101", "555.010.0101", "001 555 010 0101", "5550100101"} {
		if got, ok := NormalizePhone(raw); !ok || got != want {
			t.Errorf("%q: got %q", raw, got)
		}
	}
	for _, bad := range []string{"", "12345", "0000000000", "1111111", "1234567890", "n/a"} {
		if _, ok := NormalizePhone(bad); ok {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestNames(t *testing.T) {
	if got := strings.Join(NameTokens("Dr. José  Smith, Jr."), " "); got != "jose smith" {
		t.Errorf("got %q", got)
	}
	similar := [][2]string{{"Robert Jones", "Bob Jones"}, {"J Smith", "John Smith"}, {"Jon Smithe", "Jon Smith"}, {"Smith, John", "John Smith"}, {"José Silva", "Jose Silva"}}
	for _, pair := range similar {
		if !similarNames(NameTokens(pair[0]), NameTokens(pair[1])) {
			t.Errorf("%v should be similar", pair)
		}
	}
	different := [][2]string{{"John Smith", "Jane Smith"}, {"Sam Lee", "Priya Nair"}, {"Ana", "Ana Silva"}, {"", "Ana"}, {"Mark Lee", "Mary Lee"}}
	for _, pair := range different {
		if similarNames(NameTokens(pair[0]), NameTokens(pair[1])) {
			t.Errorf("%v should differ", pair)
		}
	}
}

func TestCompanyKey(t *testing.T) {
	if CompanyKey("ACME Limited") != CompanyKey("Acme Ltd.") || CompanyKey("Acme") == CompanyKey("Apex") {
		t.Error("company keys are wrong")
	}
}

func contact(row int, name, email, phone, company string) Contact {
	c := Contact{Row: row, Name: name, Company: company}
	if email != "" {
		c.Emails = []string{email}
	}
	if phone != "" {
		c.Phones = []string{phone}
	}
	return c
}

func onlyGroup(t *testing.T, r Result) Group {
	t.Helper()
	if len(r.Groups) != 1 {
		t.Fatalf("want one group, got %+v", r)
	}
	return r.Groups[0]
}

func TestSameEmailIsSamePerson(t *testing.T) {
	g := onlyGroup(t, Find([]Contact{contact(2, "Ana Silva", "ana+x@example.com", "", ""), contact(3, "Silva, Ana", "ANA@example.com", "", "")}))
	if g.Level != LevelSame || !slices.Equal(g.Rows, []int{2, 3}) {
		t.Errorf("got %+v", g)
	}
}

func TestSameEmailDifferentNamesIsOnlyReview(t *testing.T) {
	r := Find([]Contact{contact(2, "Lena Fischer", "shared@example.com", "", ""), contact(3, "Omar Haddad", "shared@example.com", "", "")})
	if len(r.Groups) != 0 || len(r.Review) != 1 || !strings.Contains(r.Review[0].Reason, "shared address") {
		t.Errorf("got %+v", r)
	}
}

func TestSharedMailboxNeverLinks(t *testing.T) {
	r := Find([]Contact{contact(2, "Sam Lee", "info@example.com", "", ""), contact(3, "Priya Nair", "info@example.com", "", "")})
	if len(r.Groups) != 0 || len(r.Review) != 0 {
		t.Errorf("got %+v", r)
	}
}

func TestPhone(t *testing.T) {
	g := onlyGroup(t, Find([]Contact{contact(2, "Robert Jones", "", "555-010-0102", ""), contact(3, "Bob Jones", "", "(555) 010-0102", "")}))
	if g.Level != LevelSame {
		t.Errorf("got %+v", g)
	}
	g = onlyGroup(t, Find([]Contact{contact(2, "Robert Jones", "", "555-010-0102", ""), contact(3, "", "", "555-010-0102", "")}))
	if g.Level != LevelLikely {
		t.Errorf("got %+v", g)
	}
	r := Find([]Contact{contact(2, "Robert Jones", "", "555-010-0102", ""), contact(3, "Dana Wu", "", "555-010-0102", "")})
	if len(r.Groups) != 0 || len(r.Review) != 1 {
		t.Errorf("got %+v", r)
	}
}

func TestSameNameNeedsSupport(t *testing.T) {
	g := onlyGroup(t, Find([]Contact{contact(2, "Maria Costa", "a@example.net", "", "Cafe Ltd"), contact(3, "Maria Costa", "b@other.example", "", "Cafe")}))
	if g.Level != LevelLikely {
		t.Errorf("got %+v", g)
	}
	r := Find([]Contact{contact(2, "John Smith", "", "", "Acme"), contact(3, "John Smith", "", "", "Apex")})
	if len(r.Groups) != 0 || r.Review[0].Reason != "same name, different companies" {
		t.Errorf("got %+v", r)
	}
	r = Find([]Contact{contact(2, "John Smith", "a@gmail.com", "", ""), contact(3, "John Smith", "b@gmail.com", "", "")})
	if len(r.Groups) != 0 || r.Review[0].Reason != "same name only" {
		t.Errorf("free email domains aren't evidence: %+v", r)
	}
	r = Find([]Contact{contact(2, "Madonna", "", "", ""), contact(3, "Madonna", "", "", "")})
	if len(r.Groups) != 0 || len(r.Review) != 0 {
		t.Errorf("one word isn't enough: %+v", r)
	}
}

func TestSimilarNameAtSameCompanyIsReview(t *testing.T) {
	r := Find([]Contact{contact(2, "Jon Smith", "", "", "Acme"), contact(3, "John Smith", "", "", "Acme Ltd")})
	if len(r.Groups) != 0 || len(r.Review) != 1 {
		t.Errorf("got %+v", r)
	}
}

func TestChainsAreOneGroupAtTheWeakestLevel(t *testing.T) {
	g := onlyGroup(t, Find([]Contact{
		contact(2, "Ana Silva", "ana@example.com", "", ""),
		contact(3, "Ana Silva", "ana@example.com", "5550100104", ""),
		contact(4, "", "", "5550100104", ""),
	}))
	if g.Level != LevelLikely || len(g.Rows) != 3 {
		t.Errorf("got %+v", g)
	}
}

func TestPlaceholderValuesAreIgnored(t *testing.T) {
	list := []Contact{}
	for i := 0; i < maxKeyBlock+1; i++ {
		list = append(list, contact(i+2, fmt.Sprintf("Person %d", i), "same@example.com", "", ""))
	}
	r := Find(list)
	if len(r.Groups) != 0 || len(r.Warnings) != 1 {
		t.Errorf("groups %d, warnings %v", len(r.Groups), r.Warnings)
	}
}

func TestRandomDataInvariants(t *testing.T) {
	random := rand.New(rand.NewSource(42))
	names := []string{"Ana Silva", "Bob Jones", "Robert Jones", "", "Dana Wu", "Madonna"}
	for round := 0; round < 500; round++ {
		list := []Contact{}
		for i := 0; i < random.Intn(15); i++ {
			list = append(list, contact(i+2, names[random.Intn(len(names))], []string{"", "a@example.com", "b@example.com", "info@example.com"}[random.Intn(4)], []string{"", "5550100101", "5550100102"}[random.Intn(3)], []string{"", "Acme", "Apex"}[random.Intn(3)]))
		}
		r := Find(list)
		seen := map[int]bool{}
		for _, group := range r.Groups {
			if len(group.Rows) < 2 {
				t.Fatalf("small group: %+v", group)
			}
			for _, row := range group.Rows {
				if seen[row] {
					t.Fatalf("row %d is in two groups: %+v", row, r)
				}
				seen[row] = true
			}
		}
		for _, pair := range r.Review {
			if pair.Rows[0] == pair.Rows[1] {
				t.Fatalf("pair with itself: %+v", pair)
			}
			for _, group := range r.Groups {
				if slices.Contains(group.Rows, pair.Rows[0]) && slices.Contains(group.Rows, pair.Rows[1]) {
					t.Fatalf("review pair already grouped: %+v", pair)
				}
			}
		}
		again := Find(list)
		if fmt.Sprint(r) != fmt.Sprint(again) {
			t.Fatalf("not deterministic:\n%v\n%v", r, again)
		}
	}
}

func write(t *testing.T, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "file.csv")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadFormats(t *testing.T) {
	table, err := Load(write(t, []byte("\xef\xbb\xbfExport;;\nNome;E-mail;Telefone\nAna;a@example.com;5550100101\n;;\n")), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Contacts) != 1 || table.Contacts[0].Name != "Ana" || table.Contacts[0].Row != 3 {
		t.Errorf("got %+v", table)
	}
	if !slices.Equal(table.SkippedRow, []int{4}) {
		t.Errorf("skipped %v", table.SkippedRow)
	}

	table, err = Load(write(t, []byte("First Name,Last Name,Email 1,Email 2\nAna,Silva,a@example.com;b@example.com,c@example.com\n")), nil)
	if err != nil {
		t.Fatal(err)
	}
	if table.Contacts[0].Name != "Ana Silva" || len(table.Contacts[0].Emails) != 3 {
		t.Errorf("got %+v", table.Contacts[0])
	}

	table, err = Load(write(t, []byte("Name,Email\nJos\xe9,a@example.com\n")), nil)
	if err != nil || table.Contacts[0].Name != "José" {
		t.Errorf("latin-1: %v %+v", err, table)
	}
}

func TestLoadErrors(t *testing.T) {
	path := write(t, []byte("Who,Mail\nAna,a@example.com\n"))
	if _, err := Load(path, map[string]string{"name": "Nope"}); err == nil || !strings.Contains(err.Error(), "Columns found: Who, Mail") {
		t.Errorf("got %v", err)
	}
	unknown := write(t, []byte("Foo,Bar\n1,2\n"))
	if _, err := Load(unknown, nil); err == nil || !strings.Contains(err.Error(), "First row: Foo, Bar") {
		t.Errorf("got %v", err)
	}
	if _, err := Load("/no/such/file.csv", nil); err == nil {
		t.Error("missing file should fail")
	}
	if _, err := Load(t.TempDir(), nil); err == nil {
		t.Error("folder should fail")
	}
	table, err := Load(path, map[string]string{"name": "Who", "email": "Mail"})
	if err != nil || len(table.Contacts) != 1 || len(table.Contacts[0].Emails) != 1 {
		t.Errorf("override: %v %+v", err, table)
	}
}
