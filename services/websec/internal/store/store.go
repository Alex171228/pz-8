package store

import "sync"

type Comment struct {
	Author string
	Text   string
}

type UserProfile struct {
	SessionID string
	Name      string
	CSRFToken string
}

type Store struct {
	mu       sync.RWMutex
	users    map[string]*UserProfile
	comments []Comment
}

func New() *Store {
	return &Store{
		users:    make(map[string]*UserProfile),
		comments: make([]Comment, 0),
	}
}

func (s *Store) Save(profile *UserProfile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[profile.SessionID] = profile
}

func (s *Store) Get(sessionID string) (*UserProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profile, ok := s.users[sessionID]
	return profile, ok
}

func (s *Store) UpdateName(sessionID, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, ok := s.users[sessionID]
	if !ok {
		return false
	}

	profile.Name = name
	return true
}

func (s *Store) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.users, sessionID)
}

func (s *Store) AddComment(author, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.comments = append(s.comments, Comment{Author: author, Text: text})
}

func (s *Store) GetComments() []Comment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Comment, len(s.comments))
	copy(out, s.comments)
	return out
}
