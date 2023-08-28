package raft

import "sync"

type Storage interface {
	Set(key string, value []byte)

	Get(key string) ([]byte, bool)

	StorageExists() bool
}

// Just for testing
type KVStorage struct {
	mu sync.Mutex
	m  map[string][]byte
}

func NewMapStorage() *KVStorage {
	m := make(map[string][]byte)
	return &KVStorage{
		m: m,
	}
}

func (s *KVStorage) Get(key string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.m[key]
	return v, ok
}

func (s *KVStorage) Set(key string, value []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = value
}

func (s *KVStorage) StorageExists() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.m) > 0
}
