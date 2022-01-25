package store

import (
	"errors"
	"os"
	"strings"

	"github.com/emersion/go-imap/backend"
)

type User struct {
	username    string
	password    string
	rootPath    string
	rootMailbox *Mailbox
}

// NewUser creates a new User, with messages stored under the given dir.
// name and pwd are the login name and password respectively.
func NewUser(dir, name, pwd string) (*User, error) {
	u := &User{
		rootPath: dir,
		username: name,
		password: pwd,
	}

	// Users have a single anonymous root mailbox to simplify the heirarchy.
	root, err := NewMailbox(u, nil, "")
	if err != nil {
		return nil, err
	}
	u.rootMailbox = root
	return u, err
}

func (u *User) Username() string {
	return u.username
}

func (u *User) ListMailboxes(onlySubscribed bool) ([]backend.Mailbox, error) {

	matching := []backend.Mailbox{}

	var collect func(mbox *Mailbox)
	collect = func(mbox *Mailbox) {
		if mbox.parent != nil {
			if mbox.Subscribed || !onlySubscribed {
				matching = append(matching, mbox)
			}
		}
		for _, child := range mbox.children {
			collect(child)
		}
	}

	collect(u.rootMailbox)
	return matching, nil
}

func (u *User) GetMailbox(name string) (backend.Mailbox, error) {
	return u.findMailbox(name)
}

func (u *User) CreateMailbox(name string) error {
	_, err := u.findMailbox(name)
	if err == nil {
		return errors.New("Mailbox already exists")
	}

	// create missing mailboxes, starting at the top.
	parts := strings.Split(name, Delimiter)
	if name == "" || len(parts) < 1 {
		return errors.New("Bad name")
	}
	parent := u.rootMailbox
	for i := 0; i < len(parts); i++ {
		parentName := strings.Join(parts[:i+1], Delimiter)
		mbox, err := u.findMailbox(parentName)
		if err != nil {
			mbox, err = NewMailbox(u, parent, parts[i])
			if err != nil {
				return err
			}
			parent.children = append(parent.children, mbox)
		}
		parent = mbox
	}

	return nil
}

func (u *User) DeleteMailbox(name string) error {
	if name == "INBOX" {
		return errors.New("Cannot delete INBOX")
	}
	mbox, err := u.findMailbox(name)
	if err != nil {
		return err
	}

	if len(mbox.children) > 0 {
		return errors.New("[HASCHILDREN] Has child mailboxes.")
	}

	// TODO: could handle deleting mailboxes with children - delete the
	// email files, set the \Noselect flag and leave the dir intact?

	err = os.RemoveAll(mbox.fsPath())
	if err != nil {
		return err
	}

	p := mbox.parent
	c := []*Mailbox{}
	for _, m := range p.children {
		if m == mbox {
			continue
		}
		c = append(c, m)
	}
	mbox.parent = nil
	p.children = c
	return nil
}

func (u *User) RenameMailbox(existingName, newName string) error {
	if existingName == "INBOX" {
		return errors.New("Nope.")
	}

	mbox, err := u.findMailbox(existingName)
	if err != nil {
		return err
	}

	_, err = u.findMailbox(newName)
	if err == nil {
		return errors.New("Mailbox already exists")
	}

	// create any missing intermediate mailboxes
	// (TODO: share logic with CreateMailbox()?)
	parts := strings.Split(newName, Delimiter)
	parent := u.rootMailbox
	for i := 0; i < len(parts)-1; i++ {
		parentName := strings.Join(parts[:i+1], Delimiter)
		intermediate, err := u.findMailbox(parentName)
		if err != nil {
			intermediate, err = NewMailbox(u, parent, parts[i])
			if err != nil {
				return err
			}
			parent.children = append(parent.children, intermediate)
		}
		parent = intermediate
	}

	// the main event. Rename and reparent.

	// backup
	old := *mbox

	mbox.parent = parent
	mbox.fsName = parts[len(parts)-1]
	err = os.Rename(old.fsPath(), mbox.fsPath())
	if err != nil {
		*mbox = old //restore
		return err
	}

	// reparent
	parent.children = append(parent.children, mbox)
	c := []*Mailbox{}
	for _, m := range old.parent.children {
		if m == mbox {
			continue
		}
		c = append(c, m)
	}
	old.parent.children = c
	return nil
}

func (u *User) Logout() error {
	return nil
}

// Find mailbox by IMAP name (eg 'INBOX', 'foo/bar', etc...)
// never matches root mailbox!
func (u *User) findMailbox(name string) (*Mailbox, error) {
	parts := strings.Split(name, Delimiter)
	if len(parts) == 0 {
		return nil, backend.ErrNoSuchMailbox
	}
	mbox := u.rootMailbox
	for _, part := range parts {
		var matching *Mailbox = nil
		for _, child := range mbox.children {
			if child.fsName == part {
				matching = child
				break
			}
		}
		if matching == nil {
			return nil, backend.ErrNoSuchMailbox
		}
		mbox = matching
	}

	return mbox, nil
}
