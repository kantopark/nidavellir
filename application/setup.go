package application

import (
	"nidavellir/config"
	"nidavellir/services/store"
)

// Creates an admin account in the database if it doesn't exist
func createAdminAccount(db *store.Postgres, conf *config.Config) error {
	admins, err := db.GetAdminAppUser()
	if err != nil {
		return err
	}

	if len(admins) == 0 {
		_, err = db.AddAppUser(&store.AppUser{
			Username: conf.Acct.Username,
			Password: conf.Acct.Password,
			IsAdmin:  true,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
