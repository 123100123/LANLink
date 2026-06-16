package pairing

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
)

type Manager struct {
	mu     sync.Mutex
	token  string
	length int
}

func NewManager(length int) (*Manager, error) {
	if length <= 0 {
		length = 6
	}

	manager := &Manager{
		length: length,
	}

	if err := manager.Rotate(); err != nil {
		return nil, err
	}

	return manager, nil
}

func (m *Manager) Token() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.token
}

func (m *Manager) Validate(token string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return token != "" && token == m.token
}

func (m *Manager) Rotate() error {
	token, err := generateNumericToken(m.length)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.token = token

	return nil
}

func generateNumericToken(length int) (string, error) {
	max := int64(1)

	for i := 0; i < length; i++ {
		max *= 10
	}

	value, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return "", err
	}

	format := fmt.Sprintf("%%0%dd", length)

	return fmt.Sprintf(format, value.Int64()), nil
}
