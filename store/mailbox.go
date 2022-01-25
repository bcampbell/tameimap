package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend/backendutil"
)

var Delimiter = "/"

type Mailbox struct {
	Subscribed bool
	Messages   []*Message
	user       *User
	parent     *Mailbox // nil for root
	// Use the directory name verbatim for the component of the IMAP name.
	// In theory, the IMAP name could include special chars eg "/" which we'd
	// need to escape/unescape... but for now this is fine.
	// Empty for root mailbox.
	fsName   string
	children []*Mailbox
}

// NewMailbox creates a new mailbox, scans for messages and then recursively for
// child mailboxes.
// Will create new dir in filesystem if it doesn't already exist.
// fsName is the partial name of the mailbox (just the directory name, under the parent).
func NewMailbox(user *User, parent *Mailbox, fsName string) (*Mailbox, error) {
	mbox := &Mailbox{
		Subscribed: false,
		Messages:   []*Message{},
		user:       user,
		fsName:     fsName, // just the
		parent:     parent,
		children:   []*Mailbox{},
	}

	// Recursive scan for messages and child mailboxes
	ents, err := os.ReadDir(mbox.fsPath())
	if err != nil {
		if !os.IsNotExist(err) {
			// Create dir if it doesn't already exist.
			err = os.Mkdir(mbox.fsPath(), 0777)
			if err != nil {
				return nil, err
			}
			// Done. No need to scan an empty dir.
			return mbox, nil
		}
		return nil, err
	}

	var uid uint32 = 1
	for _, ent := range ents {
		if ent.IsDir() {
			child, err := NewMailbox(user, mbox, ent.Name())
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to init mailbox: %s\n", err)
				return nil, err
			}
			mbox.children = append(mbox.children, child)
			continue
		}
		inf, err := ent.Info()
		if err != nil {
			fmt.Fprintf(os.Stderr, "SKIPPING %s: %s\n", ent.Name(), err)
		}

		msg := &Message{
			Uid:      uid,
			Date:     inf.ModTime(),
			Size:     uint32(inf.Size()),
			filename: ent.Name(),
			mbox:     mbox,
		}
		mbox.Messages = append(mbox.Messages, msg)
		uid++
	}
	return mbox, nil
}

// fsPath() returns the full path for where this mailbox's message files are.
// (eg "/tmp/bob/INBOX")
func (mbox *Mailbox) fsPath() string {

	if mbox.parent == nil {
		// root has no dir of it's own.
		return filepath.Join(mbox.user.rootPath)
	}
	return filepath.Join(mbox.parent.fsPath(), mbox.fsName)
}

// Name() returns the full IMAP name for the mailbox ("INBOX", "stuff/todo" etc).
func (mbox *Mailbox) Name() string {
	parts := []string{}
	for p := mbox; p.parent != nil; p = p.parent {
		parts = append([]string{p.fsName}, parts...)
	}
	return strings.Join(parts, Delimiter)
}

func (mbox *Mailbox) Info() (*imap.MailboxInfo, error) {
	info := &imap.MailboxInfo{
		Delimiter: Delimiter,
		Name:      mbox.Name(),
	}
	return info, nil
}

func (mbox *Mailbox) uidNext() uint32 {
	var uid uint32
	for _, msg := range mbox.Messages {
		if msg.Uid > uid {
			uid = msg.Uid
		}
	}
	uid++
	return uid
}

func (mbox *Mailbox) flags() []string {
	flagsMap := make(map[string]bool)
	for _, msg := range mbox.Messages {
		for _, f := range msg.Flags {
			if !flagsMap[f] {
				flagsMap[f] = true
			}
		}
	}

	var flags []string
	for f := range flagsMap {
		flags = append(flags, f)
	}
	return flags
}

func (mbox *Mailbox) unseenSeqNum() uint32 {
	for i, msg := range mbox.Messages {
		seqNum := uint32(i + 1)

		seen := false
		for _, flag := range msg.Flags {
			if flag == imap.SeenFlag {
				seen = true
				break
			}
		}

		if !seen {
			return seqNum
		}
	}
	return 0
}

func (mbox *Mailbox) Status(items []imap.StatusItem) (*imap.MailboxStatus, error) {
	status := imap.NewMailboxStatus(mbox.Name(), items)
	status.Flags = mbox.flags()
	status.PermanentFlags = []string{"\\Answered", "\\Flagged", "\\Deleted", "\\Seen", "\\Draft", "\\*"}
	status.UnseenSeqNum = mbox.unseenSeqNum()

	for _, name := range items {
		switch name {
		case imap.StatusMessages:
			status.Messages = uint32(len(mbox.Messages))
		case imap.StatusUidNext:
			status.UidNext = mbox.uidNext()
		case imap.StatusUidValidity:
			status.UidValidity = 1
		case imap.StatusRecent:
			status.Recent = 0 // TODO
		case imap.StatusUnseen:
			status.Unseen = 0 // TODO
		}
	}

	return status, nil
}

func (mbox *Mailbox) SetSubscribed(subscribed bool) error {
	mbox.Subscribed = subscribed
	return nil
}

func (mbox *Mailbox) Check() error {
	return nil
}

func (mbox *Mailbox) ListMessages(uid bool, seqSet *imap.SeqSet, items []imap.FetchItem, ch chan<- *imap.Message) error {
	defer close(ch)

	for i, msg := range mbox.Messages {
		seqNum := uint32(i + 1)

		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = seqNum
		}
		if !seqSet.Contains(id) {
			continue
		}

		m, err := msg.Fetch(seqNum, items)
		if err != nil {
			continue
		}

		ch <- m
	}

	return nil
}

func (mbox *Mailbox) SearchMessages(uid bool, criteria *imap.SearchCriteria) ([]uint32, error) {
	var ids []uint32
	for i, msg := range mbox.Messages {
		seqNum := uint32(i + 1)

		ok, err := msg.Match(seqNum, criteria)
		if err != nil || !ok {
			continue
		}

		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = seqNum
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (mbox *Mailbox) CreateMessage(flags []string, date time.Time, body imap.Literal) error {
	if date.IsZero() {
		date = time.Now()
	}

	uid := mbox.uidNext()
	//filename := fmt.Sprintf("%d_uid%d", time.Now().UnixMicro(), uid)

	outFile, err := os.CreateTemp(mbox.fsPath(), fmt.Sprintf("%d_*", uid))
	if err != nil {
		return err
	}

	len, err := io.Copy(outFile, body)
	if err != nil {
		outFile.Close()
		os.Remove(outFile.Name())
		return err
	}
	msg := &Message{
		Uid:      uid,
		Date:     date,
		Size:     uint32(len),
		Flags:    flags,
		filename: filepath.Base(outFile.Name()),
		mbox:     mbox,
	}
	err = outFile.Close()
	if err != nil {
		os.Remove(outFile.Name())
		return err
	}

	mbox.Messages = append(mbox.Messages, msg)
	return nil
}

func (mbox *Mailbox) UpdateMessagesFlags(uid bool, seqset *imap.SeqSet, op imap.FlagsOp, flags []string) error {
	for i, msg := range mbox.Messages {
		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = uint32(i + 1)
		}
		if !seqset.Contains(id) {
			continue
		}

		msg.Flags = backendutil.UpdateFlags(msg.Flags, op, flags)
	}

	return nil
}

func (mbox *Mailbox) CopyMessages(uid bool, seqset *imap.SeqSet, destName string) error {
	dest, err := mbox.user.findMailbox(destName)
	if err != nil {
		return err
	}

	for i, srcMsg := range mbox.Messages {
		var id uint32
		if uid {
			id = srcMsg.Uid
		} else {
			id = uint32(i + 1)
		}
		if !seqset.Contains(id) {
			continue
		}

		// copy over the file
		// (TODO: error handling/cleanup could be a little more concise...)

		srcFile, err := os.Open(srcMsg.fsPath())
		if err != nil {
			// skip
			fmt.Fprintf(os.Stderr, "ERR: %s\n", err)
			continue
		}

		destUID := dest.uidNext()
		destFile, err := os.CreateTemp(dest.fsPath(), fmt.Sprintf("%d_*", destUID))
		if err != nil {
			// skip
			srcFile.Close()
			fmt.Fprintf(os.Stderr, "ERR: %s\n", err)
			continue
		}

		len, err := io.Copy(destFile, srcFile)
		srcFile.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERR: %s\n", err)
			destFile.Close()
			os.Remove(destFile.Name())
			continue
		}

		err = destFile.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERR: %s\n", err)
			os.Remove(destFile.Name())
			continue
		}

		// If we got this far, the file was copied, so add an entry to dest Mailbox.
		destMsg := &Message{
			Uid:      destUID,
			Date:     srcMsg.Date,
			Size:     uint32(len),
			Flags:    srcMsg.Flags,
			filename: filepath.Base(destFile.Name()),
			mbox:     dest,
		}
		dest.Messages = append(dest.Messages, destMsg)
	}

	return nil
}

func (mbox *Mailbox) Expunge() error {
	for i := len(mbox.Messages) - 1; i >= 0; i-- {
		msg := mbox.Messages[i]

		deleted := false
		for _, flag := range msg.Flags {
			if flag == imap.DeletedFlag {
				deleted = true
				break
			}
		}

		if deleted {
			os.Remove(msg.fsPath())
			mbox.Messages = append(mbox.Messages[:i], mbox.Messages[i+1:]...)
		}
	}

	return nil
}
