package backend

import (
	"errors"
	"github.com/curltech/go-colla-node/mail/entity"
	"github.com/curltech/go-colla-node/mail/service"
	backend2 "github.com/emersion/go-imap/backend"
)

const Mailbox_Inbox = "INBOX"

type MailAccount struct {
	entity.MailAccount
	Mailboxes map[string]*Mailbox
}

func (ma *MailAccount) ListMailboxes(subscribed bool) (mailboxes []backend2.Mailbox, err error) {
	mailBoxes := make([]*Mailbox, 0)
	condiBean := &Mailbox{}
	condiBean.AccountName = ma.Name
	condiBean.Subscribed = subscribed
	err = service.GetMailBoxService().Find(&mailBoxes, condiBean, "", 0, 0, "")
	if err != nil {
		for _, mailbox := range mailBoxes {
			if subscribed && !mailbox.Subscribed {
				continue
			}

			ma.Mailboxes[mailbox.BoxName] = mailbox
		}
	}
	return
}

func (ma *MailAccount) GetMailbox(name string) (backend2.Mailbox, error) {
	mailbox := &Mailbox{}
	mailbox.BoxName = name
	mailbox.AccountName = ma.Name
	var err error
	found, _ := service.GetMailBoxService().Get(mailbox, false, "", "")
	if found {
		ma.Mailboxes[mailbox.BoxName] = mailbox
		return mailbox, nil
	} else {
		err = errors.New("No such mailbox")
	}
	return mailbox, err
}

func (ma *MailAccount) CreateMailbox(name string) error {
	if _, ok := ma.Mailboxes[name]; ok {
		return errors.New("Mailbox already exists")
	}
	mailbox := &Mailbox{}
	mailbox.BoxName = name
	mailbox.AccountName = ma.Name
	service.GetMailBoxService().Insert(mailbox)
	ma.Mailboxes[name] = mailbox
	return nil
}

func (ma *MailAccount) DeleteMailbox(name string) error {
	if name == Mailbox_Inbox {
		return errors.New("Cannot delete INBOX")
	}
	mailbox := &Mailbox{}
	mailbox.BoxName = name
	mailbox.AccountName = ma.Name
	service.GetMailBoxService().Delete(mailbox, "")
	if _, ok := ma.Mailboxes[name]; !ok {
		return errors.New("No such mailbox")
	}

	delete(ma.Mailboxes, name)

	return nil
}

func (ma *MailAccount) RenameMailbox(existingName, newName string) error {
	_, ok := ma.Mailboxes[existingName]
	if !ok {
		return errors.New("No such mailbox")
	}
	mailbox := &Mailbox{}
	mailbox.BoxName = newName
	mailbox.AccountName = ma.Name
	ma.Mailboxes[newName] = mailbox

	if existingName != Mailbox_Inbox {
		mailbox = &Mailbox{}
		mailbox.BoxName = existingName
		mailbox.AccountName = ma.Name
		service.GetMailBoxService().Delete(mailbox, "")
		delete(ma.Mailboxes, existingName)
	}

	return nil
}

func (ma *MailAccount) Username() string {
	return ma.Name
}

func (ma *MailAccount) Logout() error {
	return nil
}
