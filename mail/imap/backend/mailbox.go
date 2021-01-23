package backend

import (
	"errors"
	"github.com/curltech/go-colla-node/mail/imap/entity"
	"io/ioutil"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend/backendutil"
)

var Delimiter = "/"

type Mailbox struct {
	entity.MailBox
	Messages []*MailMessage
	account  *MailAccount
}

func (mbox *Mailbox) Name() string {
	return mbox.BoxName
}

func (mbox *Mailbox) Info() (*imap.MailboxInfo, error) {
	info := &imap.MailboxInfo{
		Delimiter: Delimiter,
		Name:      mbox.BoxName,
	}
	return info, nil
}

func (mbox *Mailbox) uidNext() uint32 {
	var uid uint32
	for _, msg := range mbox.Messages {
		if uint32(msg.Id) > uid {
			uid = uint32(msg.Id)
		}
	}
	uid++
	return uid
}

func (mbox *Mailbox) flags() []string {
	flagsMap := make(map[string]bool)
	for _, msg := range mbox.Messages {
		for _, f := range strings.Split(msg.Flag, ",") {
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
		for _, flag := range strings.Split(msg.Flag, ",") {
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
	status := imap.NewMailboxStatus(mbox.BoxName, items)
	status.Flags = mbox.flags()
	status.PermanentFlags = []string{"\\*"}
	status.UnseenSeqNum = mbox.unseenSeqNum()

	for _, name := range items {
		switch name {
		case imap.StatusMessages:
			status.Messages = uint32(len(mbox.Messages))
		case imap.StatusUidNext:
			status.UidNext = uint32(mbox.uidNext())
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
			id = uint32(msg.Id)
		} else {
			id = seqNum
		}
		if !seqSet.Contains(uint32(id)) {
			continue
		}

		m, err := msg.Fetch(uint32(seqNum), items)
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
			id = uint32(msg.Id)
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

	b, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}

	m := &MailMessage{}
	m.Id = uint64(mbox.uidNext())
	m.CreateDate = &date
	m.Size = uint64(len(b))
	m.Flag = strings.Join(flags, ",")
	m.Body = b
	mbox.Messages = append(mbox.Messages, m)
	return nil
}

func (mbox *Mailbox) UpdateMessagesFlags(uid bool, seqset *imap.SeqSet, op imap.FlagsOp, flags []string) error {
	for i, msg := range mbox.Messages {
		var id uint32
		if uid {
			id = uint32(msg.Id)
		} else {
			id = uint32(i + 1)
		}
		if !seqset.Contains(id) {
			continue
		}

		msg.Flag = strings.Join(backendutil.UpdateFlags(strings.Split(msg.Flag, ","), op, flags), ",")
	}

	return nil
}

func (mbox *Mailbox) CopyMessages(uid bool, seqset *imap.SeqSet, destName string) error {
	dest, ok := mbox.account.Mailboxes[destName]
	if !ok {
		return errors.New("ErrNoSuchMailbox")
	}

	for i, msg := range mbox.Messages {
		var id uint32
		if uid {
			id = uint32(msg.Id)
		} else {
			id = uint32(i + 1)
		}
		if !seqset.Contains(id) {
			continue
		}

		msgCopy := *msg
		msgCopy.Id = uint64(dest.uidNext())
		dest.Messages = append(dest.Messages, &msgCopy)
	}

	return nil
}

func (mbox *Mailbox) Expunge() error {
	for i := len(mbox.Messages) - 1; i >= 0; i-- {
		msg := mbox.Messages[i]

		deleted := false
		for _, flag := range strings.Split(msg.Flag, ",") {
			if flag == imap.DeletedFlag {
				deleted = true
				break
			}
		}

		if deleted {
			mbox.Messages = append(mbox.Messages[:i], mbox.Messages[i+1:]...)
		}
	}

	return nil
}
