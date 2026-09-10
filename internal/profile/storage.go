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

func (s *Store) Add(server Server) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	servers, err := s.readAll()
	if err != nil {
		return err
	}

	if server.ID == "" {
		server.ID = uuid.NewString()
	}
	servers = append(servers, server)
	return s.writeAll(servers)
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
	return servers, nil
}

func (s *Store) writeAll(servers []Server) error {
	data, err := json.MarshalIndent(servers, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}
