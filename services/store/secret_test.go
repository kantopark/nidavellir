package store_test

import (
	"testing"

	"github.com/dhui/dktest"
	"github.com/stretchr/testify/require"

	. "nidavellir/services/store"
)

func TestNewSecret(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	tests := []struct {
		SourceId int
		Key      string
		Value    string
		HasError bool
	}{
		{0, "Key", "Value", true},
		{1, "", "Value", true},
		{1, "Key", "", true},
		{1, "Key", "Value", false},
	}

	for _, test := range tests {
		s, err := NewSecret(test.SourceId, test.Key, test.Value)
		if test.HasError {
			assert.Error(err)
			assert.Nil(s)
		} else {
			assert.NoError(err)
			assert.IsType(Secret{}, *s)
		}
	}
}

func TestPostgres_GetSecrets(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedSources, seedSecrets)
		assert.NoError(err)

		sourceId := 1
		secrets, err := db.GetSecrets(sourceId)

		assert.NoError(err)
		assert.Len(secrets, 1)
	})
}

func TestPostgres_AddSecret(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		_, err := newTestDb(info, seedSources, seedSecrets)
		assert.NoError(err)
	})
}

func TestPostgres_UpdateSecret(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedSources, seedSecrets)
		assert.NoError(err)

		s, err := db.GetSecret(1)
		assert.NoError(err)
		s.Value = "ABC123"

		s, err = db.UpdateSecret(s)
		assert.NoError(err)

		s2, err := db.GetSecret(1)
		assert.NoError(err)
		assert.EqualValues(s2.Value, s.Value)
	})
}

func TestPostgres_RemoveSecret(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedSources)
		assert.NoError(err)

		err = db.RemoveSecret(1)
		assert.NoError(err)
	})
}

func seedSecrets(db *Postgres) error {
	// seed secret to first source
	secret, err := NewSecret(1, "key", "value")
	if err != nil {
		return err
	}
	_, err = db.AddSecret(secret)
	return err
}
