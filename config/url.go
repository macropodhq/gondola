package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Map is a conveniency type for representing
// the query and fragment specified in a configuration pseudo-url.
type Map map[string]string

// Get returns the value for the given option, or the
// empty string if no such option was specified.
func (m Map) Get(key string) string {
	return m[key]
}

// Int returns the int value for the given option. The
// second return value is true iff the key was present
// and it could be parsed as an int.
func (m Map) Int(key string) (int, bool) {
	val, err := strconv.Atoi(m.Get(key))
	return val, err == nil
}

// String returns the options encoded as a query string.
func (m Map) String() string {
	var values []string
	for k, v := range m {
		values = append(values, fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(v)))
	}
	return strings.Join(values, "&")
}

// URL represents a pseudo URL which is used to specify
// a configuration e.g. postgres://dbname=foo user=bar password=baz.
// Systems in Gondola using this configuration type include the
// cache, the ORM and the blobstore.
// Config URLs are parsed using the following algorithm:
//	- Anything before the :// is parsed as Scheme
//	- The part from the :// until the end or the first ? is parsed as Value
//      - The part from the ? until the first # is parsed as Query.
//      - The remaining, if any, is parsed as Fragment
//	- Anything after the ? is parsed as a query string and stored in Options,
//	    with the difference that multiple values for the same parameter are
//	    not supported. Only the last one is taken into account.
type URL struct {
	Scheme   string
	Value    string
	Query    Map
	Fragment Map
}

// Parse parses the given string into a configuration URL.
func (u *URL) Parse(s string) error {
	_, err := parseURL(u, s)
	return err
}

// ValueAndQuery returns the Value with the Query, using a
// ? as a separator when the latter is nonempty.
func (u *URL) ValueAndQuery() string {
	q := u.Query.String()
	if q != "" {
		return u.Value + "?" + q
	}
	return u.Value
}

// String returns the URL as a string.
func (u *URL) String() string {
	var s string
	if u.Scheme != "" {
		s = fmt.Sprintf("%s://%s", u.Scheme, u.Value)
	} else {
		s = u.Value
	}
	if len(u.Query) > 0 {
		sep := "?"
		if strings.Contains(s, "?") {
			sep = "&"
		}
		s += sep + u.Query.String()
	}
	if len(u.Fragment) > 0 {
		s += "#" + u.Fragment.String()
	}
	return s
}

func parseURL(u *URL, s string) (*URL, error) {
	p := strings.Index(s, "://")
	if p < 0 {
		return nil, fmt.Errorf("invalid config URL %q", s)
	}
	scheme, value := s[:p], s[p+3:]
	query := make(Map)
	fragment := make(Map)
	if f := strings.Index(value, "#"); f >= 0 {
		val, err := url.ParseQuery(value[f+1:])
		if err != nil {
			return nil, err
		}
		for k, v := range val {
			fragment[k] = v[len(v)-1]
		}
		value = value[:f]
	}
	if q := strings.Index(value, "?"); q >= 0 {
		val, err := url.ParseQuery(value[q+1:])
		if err != nil {
			return nil, err
		}
		for k, v := range val {
			query[k] = v[len(v)-1]
		}
		value = value[:q]
	}
	if u == nil {
		u = &URL{}
	}
	u.Scheme = scheme
	u.Value = value
	u.Query = query
	u.Fragment = fragment
	return u, nil
}

// ParseURL parses the given string into a *URL, if possible.
func ParseURL(s string) (*URL, error) {
	return parseURL(nil, s)
}

// MustParseURL works like ParseURL, but panics if there's an error.
func MustParseURL(s string) *URL {
	u, err := ParseURL(s)
	if err != nil {
		panic(err)
	}
	return u
}
