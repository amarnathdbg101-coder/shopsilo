package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"math/big"
	"sync"
	"time"
)

type otpEntry struct {
	code      string
	expiresAt time.Time
	attempts  int
}

type OTPManager struct {
	mu          sync.RWMutex
	store       map[string]*otpEntry
	ttl         time.Duration
	maxAttempts int
	stopChan    chan struct{}
	stopOnce    sync.Once
}

func NewOTPManager(ttl time.Duration, maxAttempts int) *OTPManager {
	m := &OTPManager{
		store:       make(map[string]*otpEntry),
		ttl:         ttl,
		maxAttempts: maxAttempts,
		stopChan:    make(chan struct{}),
	}
	go m.startCleanupLoop()
	return m
}

func GenerateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", fmt.Errorf("auth: failed to generate crypto random OTP: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}

func (m *OTPManager) Create(email string) (string, error) {
	code, err := GenerateCode()
	if err != nil {
		return "", err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.store[email] = &otpEntry{
		code:      code,
		expiresAt: time.Now().Add(m.ttl),
		attempts:  0,
	}

	return code, nil
}

func (m *OTPManager) Verify(email, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, exists := m.store[email]
	if !exists {
		return ErrInvalidOTP
	}

	if time.Now().After(entry.expiresAt) {
		delete(m.store, email)
		return ErrOTPExpired
	}

	if entry.attempts >= m.maxAttempts {
		delete(m.store, email)
		return ErrMaxAttemptsExceeded
	}

	match := subtle.ConstantTimeCompare([]byte(entry.code), []byte(code)) == 1

	if !match {
		entry.attempts++
		if entry.attempts >= m.maxAttempts {
			delete(m.store, email)
			return ErrMaxAttemptsExceeded
		}
		return ErrInvalidOTP
	}

	delete(m.store, email)
	return nil
}

func (m *OTPManager) startCleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanupExpired()
		case <-m.stopChan:
			return
		}
	}
}

func (m *OTPManager) cleanupExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for email, entry := range m.store {
		if now.After(entry.expiresAt) {
			delete(m.store, email)
		}
	}
}

func (m *OTPManager) Close() {
	m.stopOnce.Do(func() {
		close(m.stopChan)
	})
}
