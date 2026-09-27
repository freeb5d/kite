package profile

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// Hysteria2 servers use protocol "hysteria2". Their extra keys follow the
// share-link query: sni, insecure ("1"), obfs ("salamander"),
// obfs-password, pinSHA256, alpn, and mport for port hopping ("20000-30000"
// or "443,5000-6000").

// parseHysteria2 reads hysteria2://auth@host:port/?sni=..#name (also hy2://).
// The port may be a hopping list ("host:443,20000-30000"); the first port
// becomes Port and the whole list goes to mport.
func parseHysteria2(link string) (Server, error) {
	rest := link[strings.Index(link, "://")+3:]
	name := ""
	if i := strings.Index(rest, "#"); i >= 0 {
		name, _ = url.PathUnescape(rest[i+1:])
		rest = rest[:i]
	}
	query := ""
	if i := strings.Index(rest, "?"); i >= 0 {
		query = rest[i+1:]
		rest = rest[:i]
	}
	rest = strings.TrimSuffix(rest, "/")
	auth := ""
	if i := strings.LastIndex(rest, "@"); i >= 0 {
		auth, _ = url.PathUnescape(rest[:i])
		rest = rest[i+1:]
	}
	host, ports := rest, ""
	if strings.HasPrefix(rest, "[") {
		if i := strings.Index(rest, "]"); i >= 0 {
			host = rest[1:i]
			ports = strings.TrimPrefix(rest[i+1:], ":")
		}
	} else if i := strings.LastIndex(rest, ":"); i >= 0 {
		host, ports = rest[:i], rest[i+1:]
	}
	if host == "" {
		return Server{}, fmt.Errorf("invalid hysteria2 link: missing host")
	}
	q, _ := url.ParseQuery(query)
	extra := queryToExtra(q)
	port := 443
	if ports != "" {
		first := strings.FieldsFunc(ports, func(r rune) bool { return r == ',' || r == '-' })
		if len(first) > 0 {
			port = atoiSafe(first[0])
		}
		if strings.ContainsAny(ports, ",-") && extra["mport"] == "" {
			extra["mport"] = ports
		}
	}
	return Server{
		Name:     nameOrDefault(name, net.JoinHostPort(host, strconv.Itoa(port))),
		Protocol: "hysteria2",
		Address:  host,
		Port:     port,
		Password: auth,
		Extra:    extra,
	}, nil
}

func hysteria2Link(s Server) string {
	u := url.URL{
		Scheme:   "hysteria2",
		Host:     net.JoinHostPort(s.Address, strconv.Itoa(s.Port)),
		Path:     "/",
		RawQuery: extraQuery(s.Extra),
		Fragment: s.Name,
	}
	if s.Password != "" {
		u.User = url.User(s.Password)
	}
	return u.String()
}
