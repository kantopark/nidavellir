package store

import (
	"strings"

	"github.com/pkg/errors"
)

type Secret struct {
	Id       int    `json:"id"`
	SourceId int    `json:"sourceId"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

func NewSecret(sourceId int, key, value string) (*Secret, error) {
	s := &Secret{
		SourceId: sourceId,
		Key:      key,
		Value:    value,
	}

	if err := s.Validate(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Secret) Validate() error {
	if s.SourceId <= 0 {
		return errors.New("source id not specified")
	}

	s.Key = strings.TrimSpace(s.Key)
	if s.Key == "" {
		return errors.New("key cannot be empty or whitespace")
	}

	s.Value = strings.TrimSpace(s.Value)
	if s.Value == "" {
		return errors.New("key cannot be empty or whitespace")
	}
	return nil
}

// Adds a secret
func (p *Postgres) AddSecret(secret *Secret) (*Secret, error) {
	secret.Id = 0
	if err := secret.Validate(); err != nil {
		return nil, err
	}

	if err := p.db.Create(secret).Error; err != nil {
		return nil, errors.Wrapf(err, "could not create secret for source id %d", secret.SourceId)
	}

	return secret, nil
}

// Gets a secret by its id
func (p *Postgres) GetSecret(id int) (*Secret, error) {
	var s Secret
	if err := p.db.First(&s, "id = ?", id).Error; err != nil {
		return nil, errors.Wrapf(err, "could not get secret record with id: %d", id)
	}
	return &s, nil
}

// Gets all secrets from source Id
func (p *Postgres) GetSecrets(sourceId int) ([]*Secret, error) {
	var s []*Secret
	if err := p.db.Find(&s, "source_id = ?", sourceId).Error; err != nil {
		return nil, errors.Wrapf(err, "could not get secret record from sourceId: %d", sourceId)
	}
	return s, nil
}

// Updates a secret's key value. The sourceId and key will uniquely identify the secret
func (p *Postgres) UpdateSecret(secret *Secret) (*Secret, error) {
	if secret.Id == 0 {
		return nil, errors.New("updated secret's id not specified")
	}

	if err := secret.Validate(); err != nil {
		return nil, err
	}

	err := p.db.
		Model(secret).
		Where("id = ?", secret.Id).
		Update(*secret).
		Error
	if err != nil {
		return nil, errors.Wrap(err, "could not update secret")
	}

	return secret, nil
}

// Removes a secret. The id will uniquely identify the secret
func (p *Postgres) RemoveSecret(id int) error {
	s, err := p.GetSecret(id)
	if err != nil {
		return err
	}

	if err := p.db.Delete(s).Error; err != nil {
		return errors.Wrap(err, "could not remove secret record")
	}

	return nil
}
