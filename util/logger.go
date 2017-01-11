package util

import (
	"fmt"
	"log"
	"os"
	"strings"
)

const prefix = "busl"
const defaultDomain = "development"

func init() {
	log.SetPrefix(fmt.Sprintf("%s source=%s pid=%v ", prefix, source(), os.Getpid()))
	log.SetFlags(0)
}

func env(key, fallback string) (val string) {
	if val = os.Getenv(key); val == "" {
		val = fallback
	}

	return val
}

// Returns the reversed domain name notation, e.g.
//
//     heroku.com => com.heroku
//     staging.heroku.com => com.heroku.staging
//
// see: http://en.wikipedia.org/wiki/Reverse_domain_name_notation
func reverseDNS(domain string) string {
	s := strings.Split(domain, ".")
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return strings.Join(s, ".")
}

func source() string {
	domain := env("HEROKU_DOMAIN", defaultDomain)
	return reverseDNS(fmt.Sprintf("%s.%s", prefix, domain))
}
