package session

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"
)

type Store struct {
	mu         sync.RWMutex
	name       string
	id         string
	handler    Handler
	attributes map[string]interface{}
}

func NewStore(name string, handler Handler) *Store {
	s := &Store{
		name:       name,
		handler:    handler,
		attributes: make(map[string]interface{}),
	}
	return s
}

func (s *Store) GetHandler() Handler {
	return s.handler
}

func (s *Store) GetId() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.id
}

func (s *Store) SetId(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(id) < 1 {
		id = generateSessionId()
	}
	s.id = id
}

func (s *Store) GetName() string {
	return s.name
}

func (s *Store) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.id == "" {
		s.id = generateSessionId()
	}

	data := s.handler.Read(s.id)
	if data == "" {
		return
	}

	decodeData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return
	}

	udata := make(map[string]interface{})
	if err := json.Unmarshal(decodeData, &udata); err == nil {
		for k, v := range udata {
			s.attributes[k] = v
		}
	}
}

func (s *Store) Get(name string, value ...interface{}) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if v, ok := s.attributes[name]; ok {
		return v
	}
	if len(value) > 0 {
		return value[0]
	}
	return nil
}

func (s *Store) Set(name string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.attributes == nil {
		s.attributes = make(map[string]interface{})
	}
	s.attributes[name] = value
}

func (s *Store) All() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range s.attributes {
		result[k] = v
	}
	return result
}

func (s *Store) Remove(name string) interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	value := s.attributes[name]
	delete(s.attributes, name)
	return value
}

func (s *Store) Forget(names ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, name := range names {
		delete(s.attributes, name)
	}
}

func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.attributes = make(map[string]interface{})
}

func (s *Store) Save() {
	s.mu.RLock()
	data, err := json.Marshal(s.attributes)
	id := s.id
	handler := s.handler
	s.mu.RUnlock()

	if err != nil {
		return
	}

	encodeData := base64.StdEncoding.EncodeToString(data)
	handler.Write(id, encodeData)
}

func generateSessionId() string {
	id := strconv.FormatInt(time.Now().UnixNano(), 10)
	b := make([]byte, 48)
	_, _ = io.ReadFull(rand.Reader, b)
	id = id + base64.URLEncoding.EncodeToString(b)

	h := sha1.New()
	h.Write([]byte(id))

	return fmt.Sprintf("%x", h.Sum(nil))
}

