package backend

import (
	"bufio"
	"bytes"
	"github.com/curltech/go-colla-node/mail/imap/entity"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend/backendutil"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/textproto"
	"io"
	"strings"
)

type MailMessage struct {
	entity.MailMessage
}

func (m *MailMessage) entity() (*message.Entity, error) {
	return message.Read(bytes.NewReader(m.Body))
}

func (m *MailMessage) headerAndBody() (textproto.Header, io.Reader, error) {
	body := bufio.NewReader(bytes.NewReader(m.Body))
	hdr, err := textproto.ReadHeader(body)
	return hdr, body, err
}

func (m *MailMessage) Fetch(seqNum uint32, items []imap.FetchItem) (*imap.Message, error) {
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
			fetched.Flags = strings.Split(m.Flag, ",")
		case imap.FetchInternalDate:
			fetched.InternalDate = *m.CreateDate
		case imap.FetchRFC822Size:
			fetched.Size = uint32(m.Size)
		case imap.FetchUid:
			fetched.Uid = uint32(m.Id)
		default:
			section, err := imap.ParseBodySectionName(item)
			if err != nil {
				break
			}

			body := bufio.NewReader(bytes.NewReader(m.Body))
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

func (m *MailMessage) Match(seqNum uint32, c *imap.SearchCriteria) (bool, error) {
	e, _ := m.entity()
	return backendutil.Match(e, seqNum, uint32(m.Id), *m.CreateDate, strings.Split(m.Flag, ","), c)
}
