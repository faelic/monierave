package api

import (
	"bufio"
	"context"
	"crypto/sha1"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	minPasswordRunes = 8
	maxPasswordBytes = 72
	hibpCacheSize    = 1024
)

type PasswordBreachChecker interface {
	IsCompromised(ctx context.Context, password string) (bool, error)
}

type noopPasswordBreachChecker struct{}

func (noopPasswordBreachChecker) IsCompromised(context.Context, string) (bool, error) {
	return false, nil
}

type hibpCacheEntry struct {
	suffixes  map[string]struct{}
	expiresAt time.Time
}

type HIBPPasswordBreachChecker struct {
	baseURL string
	client  *http.Client
	ttl     time.Duration

	mu      sync.Mutex
	entries map[string]hibpCacheEntry
	order   []string
}

func NewHIBPPasswordBreachChecker(
	baseURL string,
	timeout time.Duration,
	cacheTTL time.Duration,
) *HIBPPasswordBreachChecker {
	return &HIBPPasswordBreachChecker{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
		ttl:     cacheTTL,
		entries: make(map[string]hibpCacheEntry),
	}
}

func (checker *HIBPPasswordBreachChecker) IsCompromised(
	ctx context.Context,
	password string,
) (bool, error) {
	// SHA-1 is required by the HIBP k-anonymity range protocol; the password is
	// never transmitted and the full digest is never logged.
	sum := sha1.Sum([]byte(password))
	digest := strings.ToUpper(fmt.Sprintf("%x", sum))
	prefix, suffix := digest[:5], digest[5:]

	suffixes, ok := checker.cached(prefix)
	if !ok {
		var err error
		suffixes, err = checker.fetch(ctx, prefix)
		if err != nil {
			return false, err
		}
		checker.store(prefix, suffixes)
	}
	_, compromised := suffixes[suffix]
	return compromised, nil
}

func (checker *HIBPPasswordBreachChecker) fetch(
	ctx context.Context,
	prefix string,
) (map[string]struct{}, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		checker.baseURL+"/"+prefix,
		nil,
	)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Add-Padding", "true")
	request.Header.Set("User-Agent", "monierave-password-policy")

	response, err := checker.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("query HIBP password range: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HIBP returned status %d", response.StatusCode)
	}

	suffixes := make(map[string]struct{})
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)
	for scanner.Scan() {
		value, countValue, found := strings.Cut(strings.TrimSpace(scanner.Text()), ":")
		if !found || len(value) != 35 {
			return nil, fmt.Errorf("invalid HIBP range response")
		}
		count, err := strconv.ParseUint(countValue, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid HIBP range count: %w", err)
		}
		// Add-Padding adds random zero-count suffixes to hide response size.
		if count > 0 {
			suffixes[strings.ToUpper(value)] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read HIBP range response: %w", err)
	}
	return suffixes, nil
}

func (checker *HIBPPasswordBreachChecker) cached(
	prefix string,
) (map[string]struct{}, bool) {
	checker.mu.Lock()
	defer checker.mu.Unlock()
	entry, ok := checker.entries[prefix]
	if !ok || time.Now().After(entry.expiresAt) {
		if ok {
			delete(checker.entries, prefix)
			checker.removeFromOrder(prefix)
		}
		return nil, false
	}
	return entry.suffixes, true
}

func (checker *HIBPPasswordBreachChecker) removeFromOrder(prefix string) {
	for index, value := range checker.order {
		if value == prefix {
			checker.order = append(checker.order[:index], checker.order[index+1:]...)
			return
		}
	}
}

func (checker *HIBPPasswordBreachChecker) store(
	prefix string,
	suffixes map[string]struct{},
) {
	checker.mu.Lock()
	defer checker.mu.Unlock()
	if _, exists := checker.entries[prefix]; !exists {
		checker.order = append(checker.order, prefix)
	}
	checker.entries[prefix] = hibpCacheEntry{
		suffixes:  suffixes,
		expiresAt: time.Now().Add(checker.ttl),
	}
	for len(checker.entries) > hibpCacheSize && len(checker.order) > 0 {
		oldest := checker.order[0]
		checker.order = checker.order[1:]
		delete(checker.entries, oldest)
	}
}

func validateNewPassword(password string) error {
	if utf8.RuneCountInString(password) < minPasswordRunes ||
		len([]byte(password)) > maxPasswordBytes {
		return ErrInvalidPasswordLength
	}
	return nil
}
