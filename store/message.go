package store

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend/backendutil"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/textproto"
)

type Message struct {
	Uid   uint32
	Date  time.Time
	Size  uint32
	Flags []string
	//Body     []byte
	filename string
	mbox     *Mailbox
}

func (m *Message) fsPath() string {
	return filepath.Join(m.mbox.fsPath(), m.filename)
}

func (m *Message) entity() (*message.Entity, error) {
	f, err := os.Open(m.fsPath())
	if err != nil {
		return nil, err
	}

	return message.Read(f)
}

func (m *Message) headerAndBody() (textproto.Header, io.Reader, error) {
	f, err := os.Open(m.fsPath())
	if err != nil {
		return textproto.Header{}, nil, err
	}
	body := bufio.NewReader(f)
	hdr, err := textproto.ReadHeader(body)
	return hdr, body, err
}

func (m *Message) Fetch(seqNum uint32, items []imap.FetchItem) (*imap.Message, error) {
	fetched := imap.NewMessage(seqNum, items)
	for _, item := range items {
		switch item {
		case imap.FetchEnvelope:
			hdr, _, _ := m.headerAndBody()
			fetched.Envelope, _ = backendutil.FetchEnvelope(hdr)
		case imap.FetchBody, imap.FetchBodyStructure:
			hdr, body, _ := m.headerAndBody()
			fetched.BodyStructure, _ = backendutil.FetchBodyStructure(hdr, body, item == imap.FetchBodyStructure)
		case imap.FetchFlags:
			fetched.Flags = m.Flags
		case imap.FetchInternalDate:
			fetched.InternalDate = m.Date
		case imap.FetchRFC822Size:
			fetched.Size = m.Size
		case imap.FetchUid:
			fetched.Uid = m.Uid
		default:
			section, err := imap.ParseBodySectionName(item)
			if err != nil {
				break
			}

			f, err := os.Open(m.fsPath())
			if err != nil {
				return nil, err
			}
			body := bufio.NewReader(f)
			hdr, err := textproto.ReadHeader(body)
			if err != nil {
				return nil, err
			}

			l, _ := backendutil.FetchBodySection(hdr, body, section)
			fetched.Body[section] = l
		}
	}

	return fetched, nil
}

func (m *Message) Match(seqNum uint32, c *imap.SearchCriteria) (bool, error) {
	e, _ := m.entity()
	return backendutil.Match(e, seqNum, m.Uid, m.Date, m.Flags, c)
}
