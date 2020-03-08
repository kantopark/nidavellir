package store_test

import (
	"testing"

	"github.com/dhui/dktest"
	"github.com/stretchr/testify/require"

	. "nidavellir/services/store"
)

func TestNewAccount(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	tests := []struct {
		Username string
		Password string
		IsAdmin  bool
		HasError bool
	}{
		{"username", "password", false, false},
		{"username", "", false, false},
		{"username", "", true, true},
		{"", "password", false, true},
	}

	for _, test := range tests {
		u, err := NewAccount(test.Username, test.Password, test.IsAdmin)
		if test.HasError {
			assert.Error(err)
			assert.Nil(u)
		} else {
			assert.NoError(err)
			assert.IsType(&Account{}, u)
			assert.True(u.HasValidPassword(test.Password))
			assert.False(u.HasValidPassword(test.Password + "1"))
		}
	}
}

func TestPostgres_AddAccount(t *testing.T) {
	t.Parallel()

	assert := require.New(t)
	accounts, err := newAccounts()
	assert.NoError(err)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info)
		assert.NoError(err)

		for _, u := range accounts {
			u, err := db.AddAccount(u)
			assert.NoError(err)
			assert.IsType(&Account{}, u)
		}

		// all these should lead to errors
		newAccounts, err := newAccounts()
		assert.NoError(err)
		user1 := newAccounts[0]
		_, err = db.AddAccount(user1)
		assert.Error(err, "username already exists")

		_, err = db.AddAccount(&Account{
			Username: "SomeName",
			Password: "",
		})
		assert.NoError(err, "")

	})
}

func TestPostgres_GetAdminAccounts(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts)
		assert.NoError(err)

		admins, err := db.GetAdminAccounts()
		assert.NoError(err)
		assert.Len(admins, 1)
	})
}

func TestPostgres_GetAccount(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts)
		assert.NoError(err)

		u, err := db.GetAccount("admin")
		assert.NoError(err)
		assert.IsType(&Account{}, u)

		u, err = db.GetAccount("name does not exist")
		assert.Error(err)
		assert.Nil(u)
	})
}

func TestPostgres_GetAccounts(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts)
		assert.NoError(err)

		accounts, err := db.GetAccounts()
		assert.NoError(err)
		assert.Condition(func() bool {
			return len(accounts) > 0
		})

		for _, u := range accounts {
			assert.Empty(u.Password)
		}
	})
}

func TestPostgres_RemoveAccount(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts)
		assert.NoError(err)

		tests := []struct {
			Id       int
			HasError bool
		}{
			{0, true},
			{1, false},
		}

		for _, test := range tests {
			err = db.RemoveAccount(test.Id)
			if test.HasError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		}
	})
}

func TestPostgres_UpdateAccount(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts)
		assert.NoError(err)

		u, err := db.GetAccount("admin")
		assert.NoError(err)

		expected := &Account{
			Id:       u.Id,
			Username: "NewAdminName",
			Password: "NewPassword",
			IsAdmin:  true,
		}

		u.IsAdmin = false
		u.Username = expected.Username
		u.Password = expected.Password

		u, err = db.UpdateAccount(u)

		assert.NoError(err)
		assert.Equal(expected, u)
	})
}

func TestPostgres_IsLastAdmin(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedAccounts)
		assert.NoError(err)

		isLastAdmin, err := db.IsLastAdmin(1)
		assert.NoError(err)
		assert.True(isLastAdmin)

		// Not admin at all
		isLastAdmin, err = db.IsLastAdmin(2)
		assert.NoError(err)
		assert.False(isLastAdmin)
	})
}

func newAccounts() ([]*Account, error) {
	var accounts []*Account

	values := []struct {
		Username string
		Password string
		IsAdmin  bool
	}{
		{"admin", "password", true},
		{"user1  ", "pw2", false},
		{"user2", "pw1", false},
	}

	for _, v := range values {
		u, err := NewAccount(v.Username, v.Password, v.IsAdmin)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, u)
	}
	return accounts, nil
}

func seedAccounts(db *Postgres) error {
	accounts, err := newAccounts()
	if err != nil {
		return err
	}

	for _, u := range accounts {
		_, err := db.AddAccount(u)
		if err != nil {
			return err
		}
	}
	return nil
}
