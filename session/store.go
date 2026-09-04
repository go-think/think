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

	// Age flash data
	if f, ok := s.attributes["_flash"]; ok {
		flashMap := make(map[string][]string)
		if fm, ok := f.(map[string]interface{}); ok {
			if newFlashes, ok := fm["new"].([]interface{}); ok {
				for _, nf := range newFlashes {
					if str, ok := nf.(string); ok {
						flashMap["old"] = append(flashMap["old"], str)
					}
				}
			}
		}
		flashMap["new"] = []string{}
		s.attributes["_flash"] = flashMap
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

func (s *Store) Has(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.attributes[name]
	return ok
}

func (s *Store) Pull(name string, value ...interface{}) interface{} {
	val := s.Get(name, value...)
	s.Remove(name)
	return val
}

func (s *Store) Flash(name string, value interface{}) {
	s.Set(name, value)
	s.mu.Lock()
	defer s.mu.Unlock()

	var flashes map[string][]string
	if f, ok := s.attributes["_flash"].(map[string][]string); ok {
		flashes = f
	} else {
		flashes = map[string][]string{"new": {}, "old": {}}
	}
	flashes["new"] = append(flashes["new"], name)
	s.attributes["_flash"] = flashes
}

func (s *Store) Regenerate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.id = generateSessionId()
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

// Reflash refashions all current flash data for another request.
func (s *Store) Reflash() {
	s.mu.Lock()
	defer s.mu.Unlock()

	var flashes map[string][]string
	if f, ok := s.attributes["_flash"].(map[string][]string); ok {
		flashes = f
	} else {
		flashes = map[string][]string{"new": {}, "old": {}}
	}

	flashes["new"] = append(flashes["new"], flashes["old"]...)
	flashes["old"] = []string{}
	s.attributes["_flash"] = flashes
}

// Keep keeps specific flash keys for another request.
func (s *Store) Keep(names ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var flashes map[string][]string
	if f, ok := s.attributes["_flash"].(map[string][]string); ok {
		flashes = f
	} else {
		flashes = map[string][]string{"new": {}, "old": {}}
	}

	for _, name := range names {
		for i, oldName := range flashes["old"] {
			if oldName == name {
				flashes["old"] = append(flashes["old"][:i], flashes["old"][i+1:]...)
				break
			}
		}
		flashes["new"] = append(flashes["new"], name)
	}

	s.attributes["_flash"] = flashes
}

// Now flashes a key / value pair to the session that is only available in the current request.
func (s *Store) Now(name string, value interface{}) {
	s.Set(name, value)
	s.mu.Lock()
	defer s.mu.Unlock()

	var flashes map[string][]string
	if f, ok := s.attributes["_flash"].(map[string][]string); ok {
		flashes = f
	} else {
		flashes = map[string][]string{"new": {}, "old": {}}
	}
	flashes["old"] = append(flashes["old"], name)
	s.attributes["_flash"] = flashes
}

// Token gets the CSRF token from session.
func (s *Store) Token() string {
	token := s.Get("_token")
	if tokenStr, ok := token.(string); ok && tokenStr != "" {
		return tokenStr
	}
	return s.RegenerateToken()
}

// RegenerateToken re-generates the CSRF token.
func (s *Store) RegenerateToken() string {
	token := generateSessionId()
	s.Set("_token", token)
	return token
}

// Only gets a subset of items from session.
func (s *Store) Only(names ...string) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]interface{})
	for _, name := range names {
		if val, ok := s.attributes[name]; ok {
			result[name] = val
		}
	}
	return result
}

// Except gets all session items except a specified list.
func (s *Store) Except(names ...string) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range s.attributes {
		result[k] = v
	}
	for _, name := range names {
		delete(result, name)
	}
	return result
}

// PreviousUrl gets the previous URL from session.
func (s *Store) PreviousUrl() string {
	val := s.Get("_previous.url")
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

// SetPreviousUrl sets the previous URL in session.
func (s *Store) SetPreviousUrl(url string) {
	s.Set("_previous.url", url)
}

// Invalidate flushes the session data and regenerates the ID.
func (s *Store) Invalidate() {
	s.Clear()
	s.Regenerate()
}

// Increment increments the value of an item in the session.
func (s *Store) Increment(key string, amount ...int) int {
	step := 1
	if len(amount) > 0 {
		step = amount[0]
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	val := 0
	if current, ok := s.attributes[key]; ok {
		switch v := current.(type) {
		case int:
			val = v
		case float64:
			val = int(v)
		case int64:
			val = int(v)
		}
	}
	val += step
	s.attributes[key] = val
	return val
}

// Decrement decrements the value of an item in the session.
func (s *Store) Decrement(key string, amount ...int) int {
	step := 1
	if len(amount) > 0 {
		step = amount[0]
	}
	return s.Increment(key, -step)
}

func (s *Store) Save() {
	s.mu.RLock()
	// Clear old flash data before saving
	if f, ok := s.attributes["_flash"].(map[string][]string); ok {
		for _, oldKey := range f["old"] {
			delete(s.attributes, oldKey)
		}
		f["old"] = []string{}
	}

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

