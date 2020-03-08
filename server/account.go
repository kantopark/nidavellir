package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/pkg/errors"

	"nidavellir/services/store"
)

type IAccountStore interface {
	GetAccount(name string) (*store.Account, error)
	AddAccount(account *store.Account) (*store.Account, error)
	UpdateAccount(account *store.Account) (*store.Account, error)
	RemoveAccount(id int) error
	IsLastAdmin(id int) (bool, error)
}

type AccountHandler struct {
	DB IAccountStore
}

func (a *AccountHandler) AddAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var account *store.Account
		err := readJson(r, &account)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		account, err = a.DB.AddAccount(account)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		account.MaskSensitiveData()
		toJson(w, account)
	}
}

func (a *AccountHandler) UpdateAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload *store.Account
		err := readJson(r, &payload)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		account, err := a.DB.UpdateAccount(payload)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		toJson(w, account)
	}
}

func (a *AccountHandler) RemoveAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, errors.Wrapf(err, "invalid id: %d", id).Error(), 400)
			return
		}

		isLastAdmin, err := a.DB.IsLastAdmin(id)
		if isLastAdmin {
			http.Error(w, "Cannot remove last admin account", 400)
			return
		}

		err = a.DB.RemoveAccount(id)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		ok(w)
	}
}

func (a *AccountHandler) ValidateAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload *store.Account
		err := readJson(r, &payload)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		account, err := a.DB.GetAccount(payload.Username)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		} else if account == nil {
			http.Error(w, "user not found", 400)
			return
		} else if account.Password != payload.Password {
			http.Error(w, "invalid credentials", 400)
			return
		}

		account.MaskSensitiveData()
		toJson(w, account)
	}
}
