// Package store persists all application data in a single JSON file.
// It is deliberately dependency free: two users and a few hundred sessions
// do not need a database, and a flat file is trivial to back up.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/pedro-git-projects/gymtracker/model"
)

// UserData is everything that belongs to one person.
type UserData struct {
	Name          string
	Programs      []*model.Program
	ActiveProgram string
	Current       *model.Session
	History       model.History
}

func (u *UserData) Program(id string) *model.Program {
	for _, p := range u.Programs {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (u *UserData) Active() *model.Program {
	return u.Program(u.ActiveProgram)
}

func (u *UserData) Session(id string) *model.Session {
	for i := range u.History {
		if u.History[i].ID == id {
			return &u.History[i]
		}
	}
	return nil
}

func (u *UserData) RemoveProgram(id string) {
	for i, p := range u.Programs {
		if p.ID == id {
			u.Programs = append(u.Programs[:i], u.Programs[i+1:]...)
			break
		}
	}
	if u.ActiveProgram == id {
		u.ActiveProgram = ""
	}
}

func (u *UserData) RemoveSession(id string) {
	for i := range u.History {
		if u.History[i].ID == id {
			u.History = append(u.History[:i], u.History[i+1:]...)
			return
		}
	}
}

// Data is the root document written to disk.
type Data struct {
	Users map[string]*UserData
}

// Store guards Data with a lock and writes it back after every update.
type Store struct {
	path string
	mu   sync.RWMutex
	data Data
}

func Open(path string) (*Store, error) {
	s := &Store{path: path, data: Data{Users: map[string]*UserData{}}}
	raw, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return s, nil
	case err != nil:
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if s.data.Users == nil {
		s.data.Users = map[string]*UserData{}
	}
	return s, nil
}

// View runs fn with read access. fn must not call Update.
func (s *Store) View(fn func(*Data)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn(&s.data)
}

// Update runs fn with write access and saves afterwards. fn should validate
// its input before mutating, because a returned error does not roll back.
func (s *Store) Update(fn func(*Data) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(&s.data); err != nil {
		return err
	}
	return s.save()
}

func (s *Store) save() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".gymtracker-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, s.path)
}

// NewID returns a random identifier suitable for URLs.
func NewID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
