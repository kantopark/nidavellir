package store

import (
	"github.com/pkg/errors"

	"nidavellir/libs"
)

type AppUser struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	IsAdmin  bool   `json:"isAdmin"`
}

func NewAppUser(username, password string) (*AppUser, error) {
	u := &AppUser{
		Username: username,
		Password: password,
		IsAdmin:  false,
	}

	if err := u.Validate(); err != nil {
		return nil, err
	}

	return u, nil
}

func (u *AppUser) Validate() error {
	if libs.IsEmptyOrWhitespace(u.Username) || libs.IsEmptyOrWhitespace(u.Password) {
		return errors.New("username or password cannot be empty")
	}

	return nil
}

func (u *AppUser) HasValidPassword(password string) bool {
	return u.Password == password
}

func (p *Postgres) GetAdminAppUser() ([]*AppUser, error) {
	var users []*AppUser
	if err := p.db.Find(&users, "is_admin = ?", true).Error; err != nil {
		return nil, errors.Wrap(err, "could not get admins")
	}

	return users, nil
}

// Gets an AppUser by its username
func (p *Postgres) GetAppUser(name string) (*AppUser, error) {
	var user AppUser
	if err := p.db.First(&user, "username = ?", name).Error; err != nil {
		return nil, errors.Wrapf(err, "could not get user with username '%s'", name)
	}
	return &user, nil
}

// Gets a list of all AppUsers. Since GetAppUsers is an api that is usually called from the frontend,
// the passwords are automatically masked
func (p *Postgres) GetAppUsers() ([]*AppUser, error) {
	var users []*AppUser

	if err := p.db.Find(&users).Error; err != nil {
		return nil, errors.Wrap(err, "could not get users")
	}

	for _, u := range users {
		u.Password = ""
	}

	return users, nil
}

func (p *Postgres) getAppUserById(id int) (*AppUser, error) {
	var user AppUser
	if err := p.db.First(&user, "id = ?", id).Error; err != nil {
		return nil, errors.Wrapf(err, "could not get user with id '%d'", id)
	}
	return &user, nil
}

// Creates an AppUser. The caller should check that it is the admin calling this method
func (p *Postgres) AddAppUser(user *AppUser) (*AppUser, error) {
	if err := p.db.Create(user).Error; err != nil {
		return nil, errors.Wrap(err, "could not create new user")
	}

	return user, nil
}

// Updates a users username and password. The caller should check that it is the admin calling
// this method
func (p *Postgres) UpdateAppUser(user AppUser) (*AppUser, error) {
	if user.Id <= 0 {
		return nil, errors.New("user id must be specified")
	}

	if prev, err := p.getAppUserById(user.Id); err != nil {
		return nil, err
	} else {
		// set critical components here
		user.IsAdmin = prev.IsAdmin // non-admins cannot make themselves admins
	}

	err := p.db.
		Model(&user).
		Where("id = ?", user.Id).
		Update(user).
		Error
	if err != nil {
		return nil, errors.Wrap(err, "could not update job")
	}

	return &user, nil
}

// Removes the AppUser. The caller should check that it is the admin calling this method
func (p *Postgres) RemoveAppUser(id int) error {
	if id <= 0 {
		return errors.New("user id must be specified")
	}

	if err := p.db.First(&AppUser{}, id).Error; err != nil {
		return errors.Errorf("could not find any user with id '%d'", id)
	}

	if err := p.db.Delete(&AppUser{Id: id}).Error; err != nil {
		return errors.Wrapf(err, "error removing user with id '%d'", id)
	}

	return nil
}
