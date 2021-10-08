package backend

import (
	"errors"
	"github.com/curltech/go-colla-node/mail/entity"
	"github.com/curltech/go-colla-node/mail/service"
	"io/ioutil"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend/backendutil"
)

var Delimiter = "/"

type Mailbox struct {
	entity.MailBox
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

func (mbox *Mailbox) uidNext() uint64 {
	var uid uint64 = service.GetMailMessageService().GetSeq()

	return uid
}

func (mbox *Mailbox) flags() []string {
	flagsMap := make(map[string]bool)
	for _, f := range strings.Split(mbox.Flag, ",") {
		if !flagsMap[f] {
			flagsMap[f] = true
		}
	}

	var flags []string
	for f := range flagsMap {
		flags = append(flags, f)
	}
	return flags
}

func (mbox *Mailbox) unseenSeqNum() uint32 {
	return mbox.UnseenSeqNum
}

func (mbox *Mailbox) Status(items []imap.StatusItem) (*imap.MailboxStatus, error) {
	status := imap.NewMailboxStatus(mbox.BoxName, items)
	status.Flags = mbox.flags()
	status.PermanentFlags = []string{"\\*"}
	status.UnseenSeqNum = mbox.unseenSeqNum()

	for _, name := range items {
		switch name {
		case imap.StatusMessages:
			status.Messages = mbox.MessageNum
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

	for _, seq := range seqSet.Set {
		mailMessages := make([]*entity.MailMessage, 0)
		condiBean := &entity.MailMessage{}
		condiBean.BoxName = mbox.BoxName
		err := service.GetMailMessageService().Find(mailMessages, condiBean, "", 0, 0, "Id>=? and Id<=?", seq.Start, seq.Stop)
		if err == nil {
			for _, mailMessage := range mailMessages {
				msg := MailMessage{}
				msg.MailMessage = *mailMessage
				m, err := msg.Fetch(uint32(msg.Id), items)
				if err != nil {
					continue
				}

				ch <- m
			}
		}
	}

	return nil
}

func (mbox *Mailbox) SearchMessages(uid bool, criteria *imap.SearchCriteria) ([]uint32, error) {
	var ids []uint32
	mailMessages := make([]*entity.MailMessage, 0)
	condiBean := &entity.MailMessage{}
	condiBean.BoxName = mbox.BoxName
	err := service.GetMailMessageService().Find(mailMessages, condiBean, "", 0, 0, "")
	if err == nil {
		for _, mailMessage := range mailMessages {
			msg := MailMessage{}
			msg.MailMessage = *mailMessage
			id := uint32(msg.Id)
			ok, err := msg.Match(id, criteria)
			if err != nil || !ok {
				continue
			}
			ids = append(ids, id)
		}
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
	m.splitFlag(flags)
	m.Body = b
	m.BoxName = mbox.BoxName
	service.GetMailMessageService().Insert(m)

	return nil
}

func (mbox *Mailbox) UpdateMessagesFlags(uid bool, seqset *imap.SeqSet, op imap.FlagsOp, flags []string) error {
	for _, seq := range seqset.Set {
		mailMessages := make([]*entity.MailMessage, 0)
		condiBean := &entity.MailMessage{}
		condiBean.BoxName = mbox.BoxName
		err := service.GetMailMessageService().Find(mailMessages, condiBean, "", 0, 0, "Id>=? and Id<=?", seq.Start, seq.Stop)
		if err == nil {
			for _, mailMessage := range mailMessages {
				msg := MailMessage{}
				msg.MailMessage = *mailMessage
				rs := backendutil.UpdateFlags(msg.mergeFlag(), op, flags)
				msg.splitFlag(rs)
				service.GetMailMessageService().Update(mailMessage, nil, "")
			}
		}
	}

	return nil
}

/**
拷贝满足条件的消息到另一个收件箱
*/
func (mbox *Mailbox) CopyMessages(uid bool, seqset *imap.SeqSet, destName string) error {
	mailbox := &Mailbox{}
	mailbox.BoxName = destName
	ok, _ := service.GetMailBoxService().Get(mailbox, false, "", "")
	if !ok {
		return errors.New("ErrNoSuchMailbox")
	}

	for _, seq := range seqset.Set {
		mailMessages := make([]*entity.MailMessage, 0)
		condiBean := &entity.MailMessage{}
		condiBean.BoxName = mbox.BoxName
		err := service.GetMailMessageService().Find(mailMessages, condiBean, "", 0, 0, "Id>=? and Id<=?", seq.Start, seq.Stop)
		if err == nil {
			for _, mailMessage := range mailMessages {
				mailMessage.Id = mbox.uidNext()
				mailMessage.BoxName = destName
				service.GetMailMessageService().Insert(mailMessage)
			}
		}
	}

	return nil
}

/**
删除有删除标志的信息
*/
func (mbox *Mailbox) Expunge() error {
	msg := entity.MailMessage{}
	msg.BoxName = mbox.BoxName
	msg.DeletedFlag = imap.DeletedFlag
	service.GetMailMessageService().Delete(msg, "")

	return nil
}
