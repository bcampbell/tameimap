package store

import (
	"errors"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
	"os"
)

type Backend struct {
	users map[string]*User
}

func (be *Backend) Login(_ *imap.ConnInfo, username, password string) (backend.User, error) {
	fmt.Fprintf(os.Stderr, "login: '%s' '%s'\n", username, password)
	user, ok := be.users[username]
	if ok && user.password == password {
		return user, nil
	}

	return nil, errors.New("Bad username or password")
}

func New() (*Backend, error) {
	be := &Backend{users: map[string]*User{}}
	user, err := NewUser("./bob", "bob", "pass")
	if err != nil {
		return nil, err
	}
	be.users[user.username] = user
	return be, nil
}
