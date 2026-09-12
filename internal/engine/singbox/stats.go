package singbox

// Traffic holds cumulative up/down byte counters for the active session.
//
// Regression from the xray-core version: xray-core exposed a
// stats.Manager with per-outbound counters directly. sing-box only
// tracks per-connection traffic when its clash-api/v2ray-api
// (experimental.clash_api / experimental.v2ray_api) is enabled, via
// trafficcontrol.Manager -- wiring that up is a follow-up; for now this
// always reports zero, same as this feature's very first (pre-xray-stats)
// implementation. See README known gaps.
type Traffic struct {
	Uplink   int64 `json:"uplink"`
	Downlink int64 `json:"downlink"`
}

func (m *Manager) Traffic() Traffic {
	return Traffic{}
}
