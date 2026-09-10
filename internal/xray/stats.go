package xray

// Traffic holds cumulative up/down byte counters for the active session.
// Once the real instance is wired in, this will read from xray-core's
// StatsManager API instead of parsing stdout/logs.
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

	// TODO: query m.instance's stats manager for the real counters.
	return Traffic{}
}
