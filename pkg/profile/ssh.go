package profile

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
)

// SSH servers use protocol "ssh": Password, plus extra user, pk (private
// key), pp (key passphrase) and hk (host key fingerprint or public key).

// parseSSH reads ssh://user:password@host:port?pk=..&hk=..#name.
func parseSSH(link string) (Server, error) {
	u, err := url.Parse(link)
	if err != nil {
		return Server{}, fmt.Errorf("invalid ssh link: %w", err)
	}
	if u.Hostname() == "" {
		return Server{}, fmt.Errorf("invalid ssh link: missing host")
	}
	port := atoiSafe(u.Port())
	if port == 0 {
		port = 22
	}
	extra := queryToExtra(u.Query())
	if u.User != nil {
		extra["user"] = u.User.Username()
	}
	pw := ""
	if u.User != nil {
		pw, _ = u.User.Password()
	}
	if extra["password"] != "" && pw == "" {
		pw = extra["password"]
	}
	delete(extra, "password")
	return Server{
		Name:     nameOrDefault(u.Fragment, net.JoinHostPort(u.Hostname(), strconv.Itoa(port))),
		Protocol: "ssh",
		Address:  u.Hostname(),
		Port:     port,
		Password: pw,
		Extra:    extra,
	}, nil
}

func sshLink(s Server) string {
	extra := map[string]string{}
	for k, v := range s.Extra {
		if k != "user" {
			extra[k] = v
		}
	}
	u := url.URL{
		Scheme:   "ssh",
		Host:     net.JoinHostPort(s.Address, strconv.Itoa(s.Port)),
		RawQuery: extraQuery(extra),
		Fragment: s.Name,
	}
	user := s.Extra["user"]
	if s.Password != "" {
		u.User = url.UserPassword(user, s.Password)
	} else if user != "" {
		u.User = url.User(user)
	}
	return u.String()
}
