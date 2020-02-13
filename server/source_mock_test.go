package server_test

import (
	"github.com/pkg/errors"

	"nidavellir/services/store"
)

type MockSourceStore struct {
	db map[int]*store.Source
}

func (m *MockSourceStore) AddSource(source store.Source) (*store.Source, error) {
	if source.Id != 0 {
		return nil, errors.New("source should not have id specified")
	}

	source.Id = len(m.db) + 1
	m.db[source.Id] = &source
	return &source, nil
}

func (m *MockSourceStore) GetSource(id int) (*store.Source, error) {
	if source, exist := m.db[id]; !exist {
		return nil, errors.Errorf("no source with id %d", id)
	} else {
		return source, nil
	}
}

func (m *MockSourceStore) GetSources(_ *store.GetSourceOption) ([]*store.Source, error) {
	var sources []*store.Source
	for _, s := range m.db {
		sources = append(sources, s)
	}
	return sources, nil
}

func (m *MockSourceStore) UpdateSource(source store.Source) (*store.Source, error) {
	if _, exists := m.db[source.Id]; !exists {
		return nil, errors.New("id does not exist")
	} else {
		m.db[source.Id] = &source
	}
	return &source, nil
}

func (m *MockSourceStore) RemoveSource(id int) error {
	if _, exists := m.db[id]; !exists {
		return errors.Errorf("id %d does not exists", id)
	}
	return nil
}

func (m *MockSourceStore) GetSecrets(sourceId int) ([]*store.Secret, error) {
	var secrets []*store.Secret
	if source, exist := m.db[sourceId]; exist {
		for _, s := range source.Secrets {
			secrets = append(secrets, &s)
		}
	} else {
	}
	return secrets, nil
}

func (m *MockSourceStore) AddSecret(secret store.Secret) (*store.Secret, error) {
	source, exist := m.db[secret.SourceId]
	if !exist {
		return nil, errors.Errorf("source id '%d' does not exist", secret.SourceId)
	}

	secret.Id = len(source.Secrets) + 1
	source.Secrets = append(source.Secrets, secret)
	return &secret, nil
}

func (m *MockSourceStore) UpdateSecret(secret store.Secret) (*store.Secret, error) {
	source, exist := m.db[secret.SourceId]
	if !exist {
		return nil, errors.Errorf("source id '%d' does not exist", secret.SourceId)
	}
	for i, s := range source.Secrets {
		if s.Id == secret.Id {
			source.Secrets[i] = secret
			return &secret, nil
		}
	}

	return nil, errors.Errorf("secret id '%d' does not exist", secret.Id)
}

func (m *MockSourceStore) RemoveSecret(id int) error {
	for _, source := range m.db {
		for i, secret := range source.Secrets {
			if secret.Id == id {
				source.Secrets = append(source.Secrets[:i], source.Secrets[i+1:]...)
				return nil
			}
		}
	}
	return errors.Errorf("no secret with id '%d' found", id)
}
