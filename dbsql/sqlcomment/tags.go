package sqlcomment

import (
	"fmt"
	"maps"
	"net/url"
	"sort"
	"strings"
)

const (
	KeyDBDriver    = "db_driver"
	KeyFramework   = "framework"
	KeyApplication = "application"
	KeyRoute       = "route"
	KeyController  = "controller"
	KeyAction      = "action"
)

// Tags represents key value pairs which can be serialized into SQL comment.
// See https://google.github.io/sqlcommenter/spec/.
type Tags map[string]string

func encodeValue(v string) string {
	urlEscape := strings.ReplaceAll(url.PathEscape(v), "+", "%20")
	return fmt.Sprintf("'%s'", urlEscape)
}

func encodeKey(k string) string {
	return url.QueryEscape(k)
}

// Marshal returns the sqlcommenter encoding of t.
func (t Tags) Marshal() string {
	pairs := make([]struct{ k, v string }, 0, len(t))
	for k := range t {
		pairs = append(pairs, struct{ k, v string }{encodeKey(k), encodeValue(t[k])})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].k < pairs[j].k
	})

	var b strings.Builder

	for i, p := range pairs {
		if i > 0 {
			b.WriteByte(',')
		}

		fmt.Fprintf(&b, "%s=%s", p.k, p.v)
	}

	return b.String()
}

// Merge copies the given tags into t.
func (t Tags) Merge(tags ...Tags) Tags {
	for _, c := range tags {
		maps.Copy(t, c)
	}

	return t
}
