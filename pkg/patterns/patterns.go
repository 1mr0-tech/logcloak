package patterns

import "regexp"

type BuiltIn struct {
	Name    string
	Pattern *regexp.Regexp
}

var Library = map[string]BuiltIn{
	"email":        {Name: "email", Pattern: regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)},
	"phone-e164":   {Name: "phone-e164", Pattern: regexp.MustCompile(`\+[1-9]\d{6,14}`)},
	"phone-us":     {Name: "phone-us", Pattern: regexp.MustCompile(`(\+1[\-\s]?)?\(?\d{3}\)?[\-\s]?\d{3}[\-\s]?\d{4}`)},
	"otp-6digit":   {Name: "otp-6digit", Pattern: regexp.MustCompile(`\b[0-9]{6}\b`)},
	"credit-card":  {Name: "credit-card", Pattern: regexp.MustCompile(`\b(?:\d[ \-]?){13,19}\b`)},
	"jwt":          {Name: "jwt", Pattern: regexp.MustCompile(`eyJ[a-zA-Z0-9\-_]+\.eyJ[a-zA-Z0-9\-_]+\.[a-zA-Z0-9\-_]+`)},
	"ipv4-private": {Name: "ipv4-private", Pattern: regexp.MustCompile(`(10\.\d{1,3}\.\d{1,3}\.\d{1,3}|172\.(1[6-9]|2\d|3[01])\.\d{1,3}\.\d{1,3}|192\.168\.\d{1,3}\.\d{1,3})`)},
	"uuid":         {Name: "uuid", Pattern: regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}`)},
	"iban":         {Name: "iban", Pattern: regexp.MustCompile(`\b[A-Z]{2}[0-9]{2}[A-Z0-9]{11,30}\b`)},
	"ssn":          {Name: "ssn", Pattern: regexp.MustCompile(`\b[0-9]{3}[- ][0-9]{2}[- ][0-9]{4}\b`)},
}

func Get(name string) (BuiltIn, bool) {
	p, ok := Library[name]
	return p, ok
}
