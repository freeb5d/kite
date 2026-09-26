package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

// Store persists the server list as a single JSON file under the user's
// config directory. Sqlite is deferred until v1 needs querying beyond a
// flat list.
type Store struct {
	mu   sync.Mutex
	path string
}

func NewStore() *Store {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	kiteDir := filepath.Join(dir, "kite")
	_ = os.MkdirAll(kiteDir, 0o700)

	return &Store{path: filepath.Join(kiteDir, "servers.json")}
}

func (s *Store) List() ([]Server, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readAll()
}

func (s *Store) Get(id string) (Server, error) {
	servers, err := s.List()
	if err != nil {
		return Server{}, err
	}
	for _, srv := range servers {
		if srv.ID == id {
			return srv, nil
		}
	}
	return Server{}, fmt.Errorf("no server with id %q", id)
}

// Add persists a new server, assigning it an ID if it doesn't already
// have one, and returns the stored copy (with that ID set) so the
// caller can hand back something actually usable by Get/Connect --
// the server passed in is a value, so its own ID field is never
// visible to the caller otherwise.
func (s *Store) Add(server Server) (Server, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	servers, err := s.readAll()
	if err != nil {
		return Server{}, err
	}

	if server.ID == "" {
		server.ID = uuid.NewString()
	}
	servers = append(servers, server)
	if err := s.writeAll(servers); err != nil {
		return Server{}, err
	}
	return server, nil
}

// ReplaceWhere removes every stored server matching drop, appends added
// (assigning IDs), and writes the file once -- so importing or refreshing
// a subscription with hundreds of servers isn't hundreds of full rewrites.
func (s *Store) ReplaceWhere(drop func(Server) bool, added []Server) ([]Server, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	servers, err := s.readAll()
	if err != nil {
		return nil, err
	}
	kept := servers[:0]
	for _, srv := range servers {
		if drop == nil || !drop(srv) {
			kept = append(kept, srv)
		}
	}
	for i := range added {
		if added[i].ID == "" {
			added[i].ID = uuid.NewString()
		}
	}
	if err := s.writeAll(append(kept, added...)); err != nil {
		return nil, err
	}
	return added, nil
}

// UpdateWhere applies fn to every stored server matching match, in place,
// and writes the file once. Returns how many servers matched.
func (s *Store) UpdateWhere(match func(Server) bool, fn func(*Server)) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	servers, err := s.readAll()
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range servers {
		if match(servers[i]) {
			fn(&servers[i])
			n++
		}
	}
	if n == 0 {
		return 0, nil
	}
	return n, s.writeAll(servers)
}

// Update replaces the server with the same ID, preserving its position.
func (s *Store) Update(server Server) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	servers, err := s.readAll()
	if err != nil {
		return err
	}

	for i, srv := range servers {
		if srv.ID == server.ID {
			servers[i] = server
			return s.writeAll(servers)
		}
	}
	return fmt.Errorf("no server with id %q", server.ID)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	servers, err := s.readAll()
	if err != nil {
		return err
	}

	filtered := servers[:0]
	for _, srv := range servers {
		if srv.ID != id {
			filtered = append(filtered, srv)
		}
	}
	return s.writeAll(filtered)
}

func (s *Store) readAll() ([]Server, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return []Server{}, nil
	}
	if err != nil {
		return nil, err
	}

	var servers []Server
	if err := json.Unmarshal(data, &servers); err != nil {
		return nil, fmt.Errorf("corrupt server list at %s: %w", s.path, err)
	}

	healed := false
	for i, srv := range servers {
		if srv.ID == "" {
			servers[i].ID = uuid.NewString()
			healed = true
		}
	}
	if healed {
		if err := s.writeAll(servers); err != nil {
			return nil, err
		}
	}

	return servers, nil
}

func (s *Store) writeAll(servers []Server) error {
	data, err := json.MarshalIndent(servers, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}
