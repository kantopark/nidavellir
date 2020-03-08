package server_test

import (
	"github.com/pkg/errors"

	"nidavellir/services/store"
)

type MockAccountStore struct {
	db map[int]*store.Account
}

func (m *MockAccountStore) GetAccount(name string) (*store.Account, error) {
	for _, account := range m.db {
		if account.Username == name {
			return account, nil
		}
	}
	return nil, errors.Errorf("no account with username: %s ", name)
}

func (m *MockAccountStore) AddAccount(account *store.Account) (*store.Account, error) {
	err := account.Validate()
	if err != nil {
		return nil, err
	}

	account.Id = len(m.db) + 1
	m.db[account.Id] = account

	return account, nil
}

func (m *MockAccountStore) UpdateAccount(account *store.Account) (*store.Account, error) {
	if _, exists := m.db[account.Id]; !exists {
		return nil, errors.Errorf("no account with id: %d", account.Id)
	}

	m.db[account.Id] = account
	return account, nil
}

func (m *MockAccountStore) RemoveAccount(id int) error {
	if _, exists := m.db[id]; !exists {
		return errors.Errorf("no account with id: %d", id)
	}

	delete(m.db, id)
	return nil
}

func (m *MockAccountStore) IsLastAdmin(id int) (bool, error) {
	if a, exists := m.db[id]; !exists {
		return false, errors.Errorf("no account with id: %d", id)
	} else if !a.IsAdmin {
		return false, nil
	}

	numAdmin := 0
	for _, a := range m.db {
		if a.IsAdmin {
			numAdmin++
		}
	}
	return numAdmin == 1, nil
}
