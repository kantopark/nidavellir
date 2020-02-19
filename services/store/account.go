package store

import (
	"github.com/pkg/errors"

	"nidavellir/libs"
)

type Account struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	IsAdmin  bool   `json:"isAdmin"`
}

func NewAccount(username, password string) (*Account, error) {
	u := &Account{
		Username: username,
		Password: password,
		IsAdmin:  false,
	}

	if err := u.Validate(); err != nil {
		return nil, err
	}

	return u, nil
}

func (u *Account) Validate() error {
	if libs.IsEmptyOrWhitespace(u.Username) || libs.IsEmptyOrWhitespace(u.Password) {
		return errors.New("username or password cannot be empty")
	}

	return nil
}

func (u *Account) HasValidPassword(password string) bool {
	return u.Password == password
}

func (p *Postgres) GetAdminAccounts() ([]*Account, error) {
	var accounts []*Account
	if err := p.db.Find(&accounts, "is_admin = ?", true).Error; err != nil {
		return nil, errors.Wrap(err, "could not get admins")
	}

	return accounts, nil
}

// Gets an Account by its username
func (p *Postgres) GetAccount(name string) (*Account, error) {
	var account Account
	if err := p.db.First(&account, "username = ?", name).Error; err != nil {
		return nil, errors.Wrapf(err, "could not get account with username '%s'", name)
	}
	return &account, nil
}

// Gets a list of all Account. Since GetAccounts is an api that is usually called from the frontend,
// the passwords are automatically masked
func (p *Postgres) GetAccounts() ([]*Account, error) {
	var accounts []*Account

	if err := p.db.Find(&accounts).Error; err != nil {
		return nil, errors.Wrap(err, "could not get accounts")
	}

	for _, u := range accounts {
		u.Password = ""
	}

	return accounts, nil
}

func (p *Postgres) getAccountById(id int) (*Account, error) {
	var account Account
	if err := p.db.First(&account, "id = ?", id).Error; err != nil {
		return nil, errors.Wrapf(err, "could not get account with id '%d'", id)
	}
	return &account, nil
}

// Creates an Account. The caller should check that it is the admin calling this method
func (p *Postgres) AddAccount(account *Account) (*Account, error) {
	if err := p.db.Create(account).Error; err != nil {
		return nil, errors.Wrap(err, "could not create new account")
	}

	return account, nil
}

// Updates a account's username and password. The caller should check that it is the admin calling
// this method
func (p *Postgres) UpdateAccount(account Account) (*Account, error) {
	if account.Id <= 0 {
		return nil, errors.New("account id must be specified")
	}

	if prev, err := p.getAccountById(account.Id); err != nil {
		return nil, err
	} else {
		// set critical components here
		account.IsAdmin = prev.IsAdmin // non-admins cannot make themselves admins
	}

	err := p.db.
		Model(&account).
		Where("id = ?", account.Id).
		Update(account).
		Error
	if err != nil {
		return nil, errors.Wrap(err, "could not update account")
	}

	return &account, nil
}

// Removes the Account. The caller should check that it is the admin calling this method
func (p *Postgres) RemoveAccount(id int) error {
	if id <= 0 {
		return errors.New("account id must be specified")
	}

	if err := p.db.First(&Account{}, id).Error; err != nil {
		return errors.Errorf("could not find any account with id '%d'", id)
	}

	if err := p.db.Delete(&Account{Id: id}).Error; err != nil {
		return errors.Wrapf(err, "error removing account with id '%d'", id)
	}

	return nil
}
