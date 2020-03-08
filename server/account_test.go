package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	. "nidavellir/server"
	"nidavellir/services/store"
)

func NewAccountHandler() *AccountHandler {
	return &AccountHandler{DB: &MockAccountStore{db: map[int]*store.Account{
		1: {
			Id:       1,
			Username: "admin",
			Password: "password",
			IsAdmin:  true,
		},
		2: {
			Id:       2,
			Username: "user",
			Password: "",
			IsAdmin:  false,
		},
	}}}
}

func TestAccountHandler_AddAccount(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewAccountHandler()

	for _, test := range []struct {
		Username   string
		Password   string
		IsAdmin    bool
		StatusCode int
	}{
		{"User2", "", false, http.StatusOK},
		{"User3", "password", true, http.StatusOK},
		{"User4", "", true, http.StatusBadRequest},
	} {
		w := httptest.NewRecorder()
		var buf bytes.Buffer
		err := json.NewEncoder(&buf).Encode(&store.Account{
			Username: test.Username,
			Password: test.Password,
			IsAdmin:  test.IsAdmin,
		})
		assert.NoError(err)
		r := NewTestRequest("POST", "/", &buf, nil)
		handler.AddAccount()(w, r)
		assert.Equal(test.StatusCode, w.Code)
	}
}

func TestAccountHandler_UpdateAccount(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewAccountHandler()

	// modify old account
	account, err := handler.DB.GetAccount("admin")
	assert.NoError(err)
	account.Username = "NewAdminName"

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(account)
	assert.NoError(err)

	r := NewTestRequest("PUT", "/", &buf, nil)
	w := httptest.NewRecorder()
	handler.UpdateAccount()(w, r)

	assert.Equal(w.Code, http.StatusOK)

	var respBody *store.Account
	err = readJson(w, &respBody)
	assert.NoError(err)
	assert.IsType(&store.Account{}, respBody)
	assert.Equal(account.Username, respBody.Username)
}

func TestAccountHandler_RemoveAccount(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewAccountHandler()

	tests := []struct {
		Id         string
		StatusCode int
	}{
		{"1", http.StatusBadRequest},   // last admin
		{"999", http.StatusBadRequest}, // not exists
		{"2", http.StatusOK},           // not exists
	}

	for _, test := range tests {
		w := httptest.NewRecorder()
		r := NewTestRequest("DELETE", "/", nil, map[string]string{"id": test.Id})

		handler.RemoveAccount()(w, r)
		assert.Equal(test.StatusCode, w.Code)
	}
}

func TestAccountHandler_ValidateAccount(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewAccountHandler()

	for _, test := range []struct {
		Username   string
		Password   string
		StatusCode int
	}{
		{"admin", "password", http.StatusOK},      // admin: correct
		{"user", "", http.StatusOK},               // user: correct
		{"admin", "wrong", http.StatusBadRequest}, // admin: wrong
		{"user", "wrong", http.StatusBadRequest},  // user: wrong
		{"user2", "", http.StatusBadRequest},      // user2: not exists
	} {
		w := httptest.NewRecorder()
		var buf bytes.Buffer
		err := json.NewEncoder(&buf).Encode(&store.Account{
			Username: test.Username,
			Password: test.Password,
		})
		assert.NoError(err)
		r := NewTestRequest("POST", "/validate", &buf, nil)
		handler.ValidateAccount()(w, r)
		assert.Equal(test.StatusCode, w.Code)
	}
}
