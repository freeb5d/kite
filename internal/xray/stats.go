package xray

// Traffic holds cumulative up/down byte counters for the active session,
// read from xray-core's own stats.Manager (see Manager.Start, which
// registers these counters by enabling policy.system.statsOutboundUplink/
// Downlink in the generated config).
type Traffic struct {
	Uplink   int64 `json:"uplink"`
	Downlink int64 `json:"downlink"`
}

func (m *Manager) Traffic() Traffic {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status.State != StateRunning {
		return Traffic{}
	}

	var t Traffic
	if m.uplinkCounter != nil {
		t.Uplink = m.uplinkCounter.Value()
	}
	if m.downlinkCounter != nil {
		t.Downlink = m.downlinkCounter.Value()
	}
	return t
}
