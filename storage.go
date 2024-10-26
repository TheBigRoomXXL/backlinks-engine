package main

import (
	"database/sql"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"

	"github.com/golang/groupcache/lru"
)

// CollySQLStorage reuse the implemention of the InMemoryStorage but with more features:
//   - Limit allocation for `visited` with an LRU Cache
//   - Persiste visited sites in sqlite
type CollySQLStorage struct {
	db      *sql.DB
	visited *lru.Cache
	lock    *sync.RWMutex
	jar     *cookiejar.Jar
}

// Init initializes CollySQLStorage
func (s *CollySQLStorage) Init() error {
	//  Init in memory storage
	if s.visited == nil {
		// Each entry is 64 bits because keys are uint64 and values are struct{}.
		// So if we want to allocate N bytes, then we can have N keys
		// (I think, might be wrong, will test a some point)
		allocatedBytes := 28 * 1024 * 1024 * 1024 // 28GB
		s.visited = lru.New(allocatedBytes)
	}
	if s.lock == nil {
		s.lock = &sync.RWMutex{}
	}
	if s.jar == nil {
		s.jar, err = cookiejar.New(nil)
		if err != nil {
			return err
		}
	}

	// Init on sql storage
	if s.db == nil {
		s.db, err = sql.Open("sqlite3", "./data/colly.db")
	}
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA busy_timeout = 5000;
		PRAGMA cache_size = -20000;
		PRAGMA foreign_keys = ON;
		PRAGMA temp_store = MEMORY;
	`)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS visited (
			url_hash INTEGER KEY
		)
	`)
	if err != nil {
		return err
	}
	return nil
}

// Visited implements Storage.Visited()
func (s *CollySQLStorage) Visited(requestID uint64) error {
	s.lock.Lock()
	s.visited.Add(requestID, struct{}{})
	s.lock.Unlock()

	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO visited (url_hash) VALUES (?) ON CONFLICT DO NOTHING;`,
		int(requestID),
	)
	return err
}

// IsVisited implements Storage.IsVisited()
func (s *CollySQLStorage) IsVisited(requestID uint64) (bool, error) {
	// First check in memory cache
	s.lock.Lock()
	var exists bool
	_, exists = s.visited.Get(requestID)
	s.lock.Unlock()
	if exists {
		return exists, nil
	}

	// Then check disk
	err := s.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM visited WHERE url_hash = ? LIMIT 1);`,
		int(requestID),
	).Scan(&exists)
	return exists, err
}

// Cookies implements Storage.Cookies()
func (s *CollySQLStorage) Cookies(u *url.URL) string {
	return StringifyCookies(s.jar.Cookies(u))
}

// SetCookies implements Storage.SetCookies()
func (s *CollySQLStorage) SetCookies(u *url.URL, cookies string) {
	s.jar.SetCookies(u, UnstringifyCookies(cookies))
}

// Close implements Storage.Close()
func (s *CollySQLStorage) Close() error {
	s.db.Close()
	return nil
}

// StringifyCookies serializes list of http.Cookies to string
func StringifyCookies(cookies []*http.Cookie) string {
	// Stringify cookies.
	cs := make([]string, len(cookies))
	for i, c := range cookies {
		cs[i] = c.String()
	}
	return strings.Join(cs, "\n")
}

// UnstringifyCookies deserializes a cookie string to http.Cookies
func UnstringifyCookies(s string) []*http.Cookie {
	h := http.Header{}
	for _, c := range strings.Split(s, "\n") {
		h.Add("Set-Cookie", c)
	}
	r := http.Response{Header: h}
	return r.Cookies()
}

// ContainsCookie checks if a cookie name is represented in cookies
func ContainsCookie(cookies []*http.Cookie, name string) bool {
	for _, c := range cookies {
		if c.Name == name {
			return true
		}
	}
	return false
}
