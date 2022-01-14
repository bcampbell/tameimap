package arse

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

	/*

		user := &User{username: "username", password: "password"}
		// Managing Partners:ben@example.com,carol@example.com;
		body := "From: Dudes: alice&example.com, bob@example.org;\r\n" +
			"To: bob@example.com, contact@example.org\r\n" +
			"Subject: A little message, just for you\r\n" +
			"Date: Wed, 11 May 2016 14:31:59 +0000\r\n" +
			"Message-ID: <0000000@localhost/>\r\n" +
			"Content-Type: text/plain\r\n" +
			"\r\n" +
			"Hi there :)"

		body2 := "From: Pete <pete@silly.example>\r\n" +
			"To: A Group:Chris Jones <c@a.test>,joe@where.test,John <jdoe@one.test>;\r\n" +
			"Cc: Undisclosed recipients:;\r\n" +
			"Date: Thu, 13 Feb 1969 23:32:54 -0330\r\n" +
			"Message-ID: <testabcd.1234@silly.example>\r\n" +
			"\r\n" +
			"Testing."

		user.mailboxes = map[string]*Mailbox{
			"INBOX": {
				name: "INBOX",
				user: user,
				Messages: []*Message{
					{
						Uid:   13,
						Date:  time.Now(),
						Flags: []string{"\\Seen"},
						Size:  uint32(len(body)),
						Body:  []byte(body),
					},
					{
						Uid:   933,
						Date:  time.Now(),
						Flags: []string{},
						Size:  uint32(len(body2)),
						Body:  []byte(body2),
					},
				},
			},
			"a^b": {
				name: "a^b",
				user: user,
				Messages: []*Message{
					{
						Uid:   99,
						Date:  time.Now(),
						Flags: []string{},
						Size:  uint32(len(body)),
						Body:  []byte(body),
					},
				},
			},
			"foo\\bar": {
				name: "foo\\bar",
				user: user,
				Messages: []*Message{
					{
						Uid:   42,
						Date:  time.Now(),
						Flags: []string{},
						Size:  uint32(len(body)),
						Body:  []byte(body),
					},
				},
			},
			"wibble": {
				name: "wibble",
				user: user,
				Messages: []*Message{
					{
						Uid:   69,
						Date:  time.Now(),
						Flags: []string{},
						Size:  uint32(len(body)),
						Body:  []byte(body),
					},
				},
			},
		}
	*/
	be := &Backend{users: map[string]*User{}}
	user, err := NewUser("./bob", "bob", "pass")
	if err != nil {
		return nil, err
	}
	be.users[user.username] = user
	return be, nil
}
