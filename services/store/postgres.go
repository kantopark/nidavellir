package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/pkg/errors"
)

type Postgres struct {
	db *gorm.DB
}

type DbOption struct {
	Host     string
	Port     int
	User     string
	Password string
	DbName   string
}

func (o *DbOption) ConnectionString(mask bool) string {
	var password string
	if mask {
		password = strings.Repeat("*", len(o.Password))
	} else {
		password = o.Password
	}

	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		o.Host, o.Port, o.User, password, o.DbName)
}

func New(option *DbOption) (*Postgres, error) {
	// waiting for db to be ready
	wait := 1
	for i := 1; i < 10; i++ {
		db, err := gorm.Open("postgres", option.ConnectionString(false))
		if err == nil {
			// default db setup options
			db.SingularTable(true)
			db = db.Set("gorm:auto_preload", true)
			return &Postgres{db: db}, nil
		}
		wait += i
		time.Sleep(time.Duration(wait) * time.Second)
	}

	return nil, errors.Errorf("could not connect to database with '%s'", option.ConnectionString(true))
}
