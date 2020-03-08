package authentication

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"nidavellir/config"
	"nidavellir/services/store"
)

type IStore interface {
	GetAccount(username string) (*store.Account, error)
}

type authenticator struct {
	adminOnly bool
	configs   []config.AuthConfig
	db        IStore
}

func New(db IStore, adminOnly bool, configs ...config.AuthConfig) func(http.Handler) http.Handler {
	return authenticator{
		adminOnly: adminOnly,
		configs:   configs,
		db:        db,
	}.Next()
}

func (a *authenticator) Next() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if len(a.configs) == 0 {
			// not using any authentication
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			isValid, username, err := a.Verify(r)
			if err != nil {
				log.Print(err)
			}

			if !isValid {
				forbid(w, r)
				return
			}

			ctx := r.Context()
			context.WithValue(ctx, "username", username)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (a *authenticator) Verify(r *http.Request) (isValid bool, username string, err error) {
	for _, conf := range a.configs {
		switch strings.ToUpper(conf.Type) {
		case "BASIC":
			isValid, username, err := a.verifyBasic(r)
			if err != nil {
				return false, "", err
			} else if isValid {
				return isValid, username, nil
			}
		case "JWT":
			log.Println("Not implemented yet")
		default:
			return false, "", errors.Errorf("Unknown authentication type: %s", conf.Type)
		}
	}
	return false, "", nil
}

func (a *authenticator) verifyBasic(r *http.Request) (isValid bool, username string, err error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return false, "", errors.New("Could not get authentication credentials")
	}
	account, err := a.db.GetAccount(username)
	if err != nil {
		return false, "", err
	}

	isValid = account.HasValidPassword(password)
	if a.adminOnly && !account.IsAdmin {
		isValid = false
	}
	return isValid, account.Username, nil
}

func forbid(w http.ResponseWriter, r *http.Request) {
	log.Printf("user forbid. %+v", r)
	w.WriteHeader(http.StatusForbidden)
	_, _ = fmt.Fprint(w, "user forbidden")
}
