package contacts

import (
	"sort"
	"strings"
	"unicode"
)

var accents = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a", "å", "a", "ç", "c", "é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i", "ñ", "n", "ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o", "ø", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u", "ý", "y", "ÿ", "y", "ß", "ss",
)

func plain(text string) string {
	return accents.Replace(strings.ToLower(strings.TrimSpace(text)))
}

var freeEmailDomains = map[string]bool{
	"gmail.com": true, "googlemail.com": true, "yahoo.com": true, "hotmail.com": true, "outlook.com": true, "live.com": true,
	"msn.com": true, "icloud.com": true, "me.com": true, "aol.com": true, "proton.me": true, "protonmail.com": true,
	"gmx.com": true, "mail.com": true, "uol.com.br": true, "bol.com.br": true, "terra.com.br": true, "yahoo.com.br": true,
}

var sharedMailboxes = map[string]bool{
	"info": true, "sales": true, "contact": true, "contato": true, "admin": true, "support": true, "hello": true, "office": true,
	"billing": true, "accounts": true, "team": true, "hr": true, "noreply": true, "no-reply": true, "enquiries": true, "mail": true,
}

// NormalizeEmail returns the comparison form of an address, and false if it
// isn't one. Gmail ignores dots, and most providers ignore a +tag.
func NormalizeEmail(raw string) (string, bool) {
	email := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(raw)), "mailto:")
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 || !strings.Contains(email[at:], ".") || strings.ContainsAny(email, " ,") {
		return "", false
	}
	local, domain := email[:at], email[at+1:]
	if plus := strings.Index(local, "+"); plus > 0 {
		local = local[:plus]
	}
	if domain == "googlemail.com" {
		domain = "gmail.com"
	}
	if domain == "gmail.com" {
		local = strings.ReplaceAll(local, ".", "")
	}
	return local + "@" + domain, true
}

func emailParts(email string) (local, domain string) {
	at := strings.LastIndex(email, "@")
	return email[:at], email[at+1:]
}

// NormalizePhone keeps the last 10 digits, so +1 and 00 prefixes don't matter,
// and rejects numbers too short or too regular to identify anyone.
func NormalizePhone(raw string) (string, bool) {
	var digits strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	number := digits.String()
	if len(number) < 7 {
		return "", false
	}
	if len(number) > 10 {
		number = number[len(number)-10:]
	}
	if strings.Count(number, string(number[0])) == len(number) || strings.Contains("1234567890", number) || strings.Contains("0123456789", number) {
		return "", false
	}
	return number, true
}

var nameNoise = map[string]bool{"mr": true, "mrs": true, "ms": true, "miss": true, "dr": true, "prof": true, "sr": true, "sra": true, "jr": true, "ii": true, "iii": true}

// NameTokens lowercases, removes accents and titles, and sorts the words, so
// "Smith, John" and "Dr. John Smith" compare equal.
func NameTokens(raw string) []string {
	words := strings.FieldsFunc(plain(raw), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '\'' })
	tokens := words[:0]
	for _, word := range words {
		if !nameNoise[word] {
			tokens = append(tokens, word)
		}
	}
	sort.Strings(tokens)
	return tokens
}

var companyNoise = map[string]bool{
	"ltd": true, "limited": true, "llc": true, "inc": true, "incorporated": true, "corp": true, "corporation": true, "co": true, "company": true,
	"gmbh": true, "ltda": true, "sa": true, "plc": true, "the": true, "and": true,
}

// CompanyKey removes legal suffixes, so "Acme Ltd" and "ACME Limited" match.
func CompanyKey(raw string) string {
	var kept []string
	for _, word := range strings.FieldsFunc(plain(raw), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if !companyNoise[word] {
			kept = append(kept, word)
		}
	}
	return strings.Join(kept, " ")
}

var nicknames = map[string]string{
	"jon": "john", "johnny": "john", "bob": "robert", "rob": "robert", "bobby": "robert", "bill": "william", "will": "william", "billy": "william", "liam": "william",
	"mike": "michael", "mick": "michael", "jim": "james", "jimmy": "james", "tom": "thomas", "tommy": "thomas", "dave": "david",
	"chris": "christopher", "kate": "katherine", "katie": "katherine", "kathy": "katherine", "catherine": "katherine", "joe": "joseph",
	"ben": "benjamin", "dan": "daniel", "danny": "daniel", "matt": "matthew", "nick": "nicholas", "sam": "samuel", "tony": "anthony",
	"alex": "alexander", "steve": "steven", "stephen": "steven", "liz": "elizabeth", "beth": "elizabeth", "jen": "jennifer", "jenny": "jennifer",
	"sue": "susan", "pat": "patricia", "rick": "richard", "dick": "richard", "greg": "gregory", "andy": "andrew", "drew": "andrew",
}

func canonicalWord(word string) string {
	if full, ok := nicknames[word]; ok {
		return full
	}
	return word
}

// similarNames allows nicknames, initials ("J Smith") and small typos, but
// both names must have the same number of words.
func similarNames(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	if strings.Join(a, " ") == strings.Join(b, " ") {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	used := make([]bool, len(b))
	for _, left := range a {
		found := false
		for i, right := range b {
			if !used[i] && wordsAlike(left, right) {
				used[i], found = true, true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func wordsAlike(a, b string) bool {
	if a == b || canonicalWord(a) == canonicalWord(b) {
		return true
	}
	if (len(a) == 1 && strings.HasPrefix(b, a)) || (len(b) == 1 && strings.HasPrefix(a, b)) {
		return true
	}
	// Short names one letter apart are often different people: Mark and Mary.
	return len(a) >= 5 && len(b) >= 5 && editDistance(a, b) <= 1
}

func editDistance(a, b string) int {
	left, right := []rune(a), []rune(b)
	row := make([]int, len(right)+1)
	for j := range row {
		row[j] = j
	}
	for i := 1; i <= len(left); i++ {
		previous := row[0]
		row[0] = i
		for j := 1; j <= len(right); j++ {
			cost := 1
			if left[i-1] == right[j-1] {
				cost = 0
			}
			current := row[j]
			row[j] = min(row[j]+1, row[j-1]+1, previous+cost)
			previous = current
		}
	}
	return row[len(right)]
}
