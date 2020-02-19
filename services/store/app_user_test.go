package store_test

import (
	"testing"

	"github.com/dhui/dktest"
	"github.com/stretchr/testify/require"

	. "nidavellir/services/store"
)

func TestNewUser(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	tests := []struct {
		Username string
		Password string
		IsAdmin  bool
		HasError bool
	}{
		{"username", "password", false, false},
		{"123  ", "", false, true},
		{"", "", false, true},
	}

	for _, test := range tests {
		u, err := NewAppUser(test.Username, test.Password)
		if test.HasError {
			assert.Error(err)
			assert.Nil(u)
		} else {
			assert.NoError(err)
			assert.IsType(&AppUser{}, u)
			assert.True(u.HasValidPassword(test.Password))
			assert.False(u.HasValidPassword(test.Password + "1"))
		}
	}
}

func TestPostgres_AddAppUser(t *testing.T) {
	t.Parallel()

	assert := require.New(t)
	users, err := newUsers()
	assert.NoError(err)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info)
		assert.NoError(err)

		for _, u := range users {
			u, err := db.AddAppUser(u)
			assert.NoError(err)
			assert.IsType(&AppUser{}, u)
		}

		// all these should lead to errors
		newUsers, err := newUsers()
		assert.NoError(err)
		user1 := newUsers[0]
		_, err = db.AddAppUser(user1)
		assert.Error(err, "username already exists")

		_, err = db.AddAppUser(&AppUser{
			Username: "SomeName",
			Password: "",
		})
		assert.Error(err, "password is empty")
	})
}

func TestPostgres_GetAdminAppUser(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedUsers)
		assert.NoError(err)

		admins, err := db.GetAdminAppUser()
		assert.NoError(err)
		assert.Len(admins, 1)
	})
}

func TestPostgres_GetAppUser(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedUsers)
		assert.NoError(err)

		u, err := db.GetAppUser("admin")
		assert.NoError(err)
		assert.IsType(&AppUser{}, u)

		u, err = db.GetAppUser("name does not exist")
		assert.Error(err)
		assert.Nil(u)
	})
}

func TestPostgres_GetAppUsers(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedUsers)
		assert.NoError(err)

		users, err := db.GetAppUsers()
		assert.NoError(err)
		assert.Condition(func() bool {
			return len(users) > 0
		})

		for _, u := range users {
			assert.Empty(u.Password)
		}
	})
}

func TestPostgres_RemoveAppUser(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedUsers)
		assert.NoError(err)

		tests := []struct {
			Id       int
			HasError bool
		}{
			{0, true},
			{1, false},
		}

		for _, test := range tests {
			err = db.RemoveAppUser(test.Id)
			if test.HasError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		}
	})
}

func TestPostgres_UpdateAppUser(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	dktest.Run(t, imageName, postgresImageOptions, func(t *testing.T, info dktest.ContainerInfo) {
		db, err := newTestDb(info, seedUsers)
		assert.NoError(err)

		u, err := db.GetAppUser("admin")
		assert.NoError(err)

		expected := &AppUser{
			Id:       u.Id,
			Username: "NewAdminName",
			Password: "NewPassword",
			IsAdmin:  true,
		}

		u.IsAdmin = false
		u.Username = expected.Username
		u.Password = expected.Password

		u, err = db.UpdateAppUser(*u)

		assert.NoError(err)
		assert.Equal(expected, u)
	})
}

func newUsers() ([]*AppUser, error) {
	var users []*AppUser

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
		u, err := NewAppUser(v.Username, v.Password)
		if err != nil {
			return nil, err
		}
		u.IsAdmin = v.IsAdmin
		users = append(users, u)
	}
	return users, nil
}

func seedUsers(db *Postgres) error {
	users, err := newUsers()
	if err != nil {
		return err
	}

	for _, u := range users {
		_, err := db.AddAppUser(u)
		if err != nil {
			return err
		}
	}
	return nil
}
