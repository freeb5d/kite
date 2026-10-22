package conf

// Backported from upstream xray-core (transport_method.go) by Kite.

import (
	"encoding/json"

	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/transport/internet/xdrive"
	"google.golang.org/protobuf/proto"
)

type XDriveConfig struct {
	RemoteFolder      string          `json:"remoteFolder"`
	Service           string          `json:"service"`
	Secrets           []string        `json:"secrets"`
	SegmentBytes      uint32          `json:"segmentBytes"`
	FlushIntervalMs   uint32          `json:"flushIntervalMs"`
	PollIntervalMs    uint32          `json:"pollIntervalMs"`
	MaxPollIntervalMs uint32          `json:"maxPollIntervalMs"`
	SessionTTLSeconds uint32          `json:"sessionTtlSeconds"`
	Concurrency       uint32          `json:"concurrency"`
	EagerWindowMs     uint32          `json:"eagerWindowMs"`
	HoleTimeoutMs     uint32          `json:"holeTimeoutMs"`
	Template          json.RawMessage `json:"template"`
}

// Build implements Buildable.
func (c *XDriveConfig) Build() (proto.Message, error) {
	switch c.Service {
	case "local":
	case "Google Drive":
		if len(c.Secrets) != 3 {
			return nil, errors.New("Google Drive needs 3 secrets in order of ClientID, ClientSecret, RefreshToken")
		}
	case "template":
		if len(c.Template) == 0 {
			return nil, errors.New(`service "template" needs a "template" object`)
		}
	default:
		return nil, errors.New("unsupported service")
	}
	return &xdrive.Config{
		RemoteFolder:      c.RemoteFolder,
		Service:           c.Service,
		Secrets:           c.Secrets,
		SegmentBytes:      c.SegmentBytes,
		FlushIntervalMs:   c.FlushIntervalMs,
		PollIntervalMs:    c.PollIntervalMs,
		MaxPollIntervalMs: c.MaxPollIntervalMs,
		SessionTtlSeconds: c.SessionTTLSeconds,
		Concurrency:       c.Concurrency,
		EagerWindowMs:     c.EagerWindowMs,
		HoleTimeoutMs:     c.HoleTimeoutMs,
		Template:          string(c.Template),
	}, nil
}
