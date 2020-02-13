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

func (m MockSourceStore) GetSource(id int) (*store.Source, error) {
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
