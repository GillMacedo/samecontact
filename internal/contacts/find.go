package contacts

import (
	"fmt"
	"sort"
	"strings"
)

// Levels say how sure a group is. Only "same" and "likely" group rows together;
// "review" pairs stay separate, because they need a person to decide.
const (
	LevelSame   = "same"
	LevelLikely = "likely"
)

const (
	maxKeyBlock  = 2000
	maxNameBlock = 300
)

// Group is a set of rows that are probably one person.
type Group struct {
	Level   string   `json:"level"`
	Rows    []int    `json:"rows"`
	Names   []string `json:"names"`
	Reasons []string `json:"reasons"`
}

// Pair is two rows a person should look at.
type Pair struct {
	Rows   [2]int    `json:"rows"`
	Names  [2]string `json:"names"`
	Reason string    `json:"reason"`
}

// Result is the full output. Every contact is in at most one group.
type Result struct {
	Contacts int      `json:"contacts"`
	Groups   []Group  `json:"groups"`
	Review   []Pair   `json:"review"`
	Warnings []string `json:"warnings"`
}

// ExtraRows is how many rows would go if every group became one contact.
func (r Result) ExtraRows() int {
	total := 0
	for _, group := range r.Groups {
		total += len(group.Rows) - 1
	}
	return total
}

type person struct {
	tokens  []string
	name    string
	emails  []string
	phones  []string
	company string
	domain  string
}

type edge struct {
	a, b   int
	level  string
	reason string
}

// Find groups the contacts. It never reads anything but the given contacts.
func Find(list []Contact) Result {
	result := Result{Contacts: len(list), Groups: []Group{}, Review: []Pair{}, Warnings: []string{}}
	people := make([]person, len(list))
	blocks := map[string][]int{}
	add := func(key string, index int) { blocks[key] = append(blocks[key], index) }

	for i, contact := range list {
		p := person{tokens: NameTokens(contact.Name), company: CompanyKey(contact.Company)}
		p.name = strings.Join(p.tokens, " ")
		for _, raw := range contact.Emails {
			if email, ok := NormalizeEmail(raw); ok {
				p.emails = append(p.emails, email)
				local, domain := emailParts(email)
				if !sharedMailboxes[local] {
					add("email:"+email, i)
				}
				if !freeEmailDomains[domain] {
					p.domain = domain
				}
			}
		}
		for _, raw := range contact.Phones {
			if phone, ok := NormalizePhone(raw); ok {
				p.phones = append(p.phones, phone)
				add("phone:"+phone, i)
			}
		}
		if len(p.tokens) >= 2 {
			add("name:"+p.name, i)
		}
		if p.company != "" {
			add("company:"+p.company, i)
		}
		if p.domain != "" {
			add("domain:"+p.domain, i)
		}
		if strings.Count(contact.cells, "\x00") >= 1 {
			add("row:"+contact.cells, i)
		}
		people[i] = p
	}

	keys := make([]string, 0, len(blocks))
	for key := range blocks {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	seen := map[[2]int]map[string]bool{}
	var edges []edge
	link := func(a, b int, level, reason string) {
		pair := [2]int{a, b}
		if seen[pair] == nil {
			seen[pair] = map[string]bool{}
		}
		if !seen[pair][reason] {
			seen[pair][reason] = true
			edges = append(edges, edge{a, b, level, reason})
		}
	}

	for _, key := range keys {
		members := blocks[key]
		if len(members) < 2 {
			continue
		}
		kind, value, _ := strings.Cut(key, ":")
		limit := maxKeyBlock
		if kind == "name" || kind == "company" || kind == "domain" {
			limit = maxNameBlock
		}
		if len(members) > limit {
			if kind == "email" || kind == "phone" {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%d rows share one %s (%s), so it was ignored. It may be a placeholder.", len(members), kind, value))
			}
			continue
		}
		for x := 0; x < len(members); x++ {
			for y := x + 1; y < len(members); y++ {
				compare(kind, members[x], members[y], people, link)
			}
		}
	}

	return assemble(list, people, edges, result)
}

func compare(kind string, a, b int, people []person, link func(int, int, string, string)) {
	left, right := people[a], people[b]
	namesEqual := left.name != "" && left.name == right.name
	alike := similarNames(left.tokens, right.tokens)
	missing := left.name == "" || right.name == ""

	switch kind {
	case "row":
		link(a, b, LevelSame, "identical row")
	case "email":
		switch {
		case alike || missing:
			link(a, b, LevelSame, "same email")
		default:
			link(a, b, "review", "same email, different names: may be a shared address")
		}
	case "phone":
		switch {
		case alike:
			link(a, b, LevelSame, "same phone and matching name")
		case missing:
			link(a, b, LevelLikely, "same phone, one name missing")
		default:
			link(a, b, "review", "same phone, different names: may be a shared line")
		}
	case "name":
		switch {
		case left.company != "" && left.company == right.company:
			link(a, b, LevelLikely, "same name and company")
		case left.domain != "" && left.domain == right.domain:
			link(a, b, LevelLikely, "same name and email domain")
		case left.company != "" && right.company != "":
			// Same name at two named companies is usually two people, or a job
			// change, and only a person can tell which.
			link(a, b, "review", "same name, different companies")
		default:
			link(a, b, "review", "same name only")
		}
	case "company", "domain":
		if alike && !namesEqual {
			link(a, b, "review", "similar name at the same "+kind)
		}
	}
}

func assemble(list []Contact, people []person, edges []edge, result Result) Result {
	parent := make([]int, len(list))
	for i := range parent {
		parent[i] = i
	}
	var root func(int) int
	root = func(x int) int {
		if parent[x] != x {
			parent[x] = root(parent[x])
		}
		return parent[x]
	}
	sameParent := append([]int(nil), parent...)
	var sameRoot func(int) int
	sameRoot = func(x int) int {
		if sameParent[x] != x {
			sameParent[x] = sameRoot(sameParent[x])
		}
		return sameParent[x]
	}

	reasons := map[int]map[string]bool{}
	for _, e := range edges {
		if e.level == "review" {
			continue
		}
		parent[root(e.a)] = root(e.b)
		if e.level == LevelSame {
			sameParent[sameRoot(e.a)] = sameRoot(e.b)
		}
	}
	for _, e := range edges {
		if e.level != "review" {
			r := root(e.a)
			if reasons[r] == nil {
				reasons[r] = map[string]bool{}
			}
			reasons[r][e.reason] = true
		}
	}

	members := map[int][]int{}
	for i := range list {
		members[root(i)] = append(members[root(i)], i)
	}
	for r, group := range members {
		if len(group) < 2 {
			continue
		}
		g := Group{Level: LevelSame, Rows: []int{}, Names: []string{}, Reasons: []string{}}
		names := map[string]bool{}
		for _, index := range group {
			g.Rows = append(g.Rows, list[index].Row)
			if name := list[index].Name; name != "" && !names[name] {
				names[name] = true
				g.Names = append(g.Names, name)
			}
			if sameRoot(index) != sameRoot(group[0]) {
				g.Level = LevelLikely
			}
		}
		for reason := range reasons[r] {
			g.Reasons = append(g.Reasons, reason)
		}
		sort.Strings(g.Reasons)
		result.Groups = append(result.Groups, g)
	}
	sort.Slice(result.Groups, func(i, j int) bool {
		a, b := result.Groups[i], result.Groups[j]
		if a.Level != b.Level {
			return a.Level == LevelSame
		}
		return a.Rows[0] < b.Rows[0]
	})

	reviewed := map[[2]int]bool{}
	for _, e := range edges {
		if e.level != "review" || root(e.a) == root(e.b) || reviewed[[2]int{e.a, e.b}] {
			continue
		}
		reviewed[[2]int{e.a, e.b}] = true
		result.Review = append(result.Review, Pair{Rows: [2]int{list[e.a].Row, list[e.b].Row}, Names: [2]string{list[e.a].Name, list[e.b].Name}, Reason: e.reason})
	}
	sort.Slice(result.Review, func(i, j int) bool {
		a, b := result.Review[i].Rows, result.Review[j].Rows
		return a[0] < b[0] || (a[0] == b[0] && a[1] < b[1])
	})
	return result
}

// RowLabels maps each grouped row to its group number and level, for annotated output.
func (r Result) RowLabels() map[int][2]string {
	labels := map[int][2]string{}
	for i, group := range r.Groups {
		for _, row := range group.Rows {
			labels[row] = [2]string{fmt.Sprintf("%d", i+1), group.Level}
		}
	}
	return labels
}
