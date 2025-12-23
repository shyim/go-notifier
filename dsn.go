package notifier

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// DSN represents a Data Source Name for transport configuration.
type DSN struct {
	scheme      string
	host        string
	user        string
	password    string
	port        int
	path        string
	options     map[string]string
	originalDSN string
}

// NewDSN parses a DSN string and returns a DSN struct.
func NewDSN(dsn string) (*DSN, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid DSN: %w", err)
	}

	if u.Scheme == "" {
		return nil, fmt.Errorf("DSN must contain a scheme")
	}
	if u.Host == "" {
		return nil, fmt.Errorf("DSN must contain a host (use 'default' by default)")
	}

	options := make(map[string]string)
	if u.RawQuery != "" {
		query, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return nil, fmt.Errorf("invalid query parameters: %w", err)
		}
		for k, v := range query {
			if len(v) > 0 {
				options[k] = v[0]
			}
		}
	}

	port := 0
	if u.Port() != "" {
		p, err := strconv.Atoi(u.Port())
		if err != nil {
			return nil, fmt.Errorf("invalid port: %w", err)
		}
		port = p
	}

	password, _ := u.User.Password()
	return &DSN{
		scheme:      u.Scheme,
		host:        u.Hostname(),
		user:        u.User.Username(),
		password:    password,
		port:        port,
		path:        u.Path,
		options:     options,
		originalDSN: dsn,
	}, nil
}

func (d *DSN) GetScheme() string {
	return d.scheme
}

func (d *DSN) GetHost() string {
	return d.host
}

func (d *DSN) GetUser() string {
	return d.user
}

func (d *DSN) GetPassword() string {
	return d.password
}

func (d *DSN) GetPort(defaultPort ...int) int {
	if d.port > 0 {
		return d.port
	}
	if len(defaultPort) > 0 {
		return defaultPort[0]
	}
	return 0
}

func (d *DSN) GetOption(key string, defaultValue ...string) string {
	if val, ok := d.options[key]; ok {
		return val
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

func (d *DSN) GetRequiredOption(key string) (string, error) {
	val := d.GetOption(key)
	if val == "" {
		return "", fmt.Errorf("missing required option: %s", key)
	}
	return val, nil
}

func (d *DSN) GetBooleanOption(key string, defaultValue ...bool) bool {
	val := d.GetOption(key)
	if val == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}
	val = strings.ToLower(val)
	return val == "true" || val == "1" || val == "yes"
}

func (d *DSN) GetOptions() map[string]string {
	return d.options
}

func (d *DSN) GetPath() string {
	return d.path
}

func (d *DSN) GetOriginalDSN() string {
	return d.originalDSN
}
