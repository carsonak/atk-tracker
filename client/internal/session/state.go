package session

import "sync"

type State struct {
	mu           sync.RWMutex
	sessionID    string
	apprenticeID string
}

func (s *State) Set(sessionID, apprenticeID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionID = sessionID
	s.apprenticeID = apprenticeID
}

func (s *State) Get() (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessionID, s.apprenticeID
}

func (s *State) Clear() {
	s.Set("", "")
}
