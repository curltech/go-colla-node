// A memory backend.
package backend

import (
	"errors"
	"github.com/curltech/go-colla-node/mail/service"
	backend2 "github.com/emersion/go-imap/backend"

	"github.com/emersion/go-imap"
)

type Backend struct {
}

var backend = &Backend{}

func GetBackend() *Backend {
	return backend
}

func (be *Backend) Login(connInfo *imap.ConnInfo, username, password string) (backend2.User, error) {
	account := &MailAccount{}
	account.Name = username
	account.Password = password
	found, _ := service.GetMailAccountService().Get(account, false, "", "")
	if found {
		return account, nil
	}

	return nil, errors.New("Bad name or password")
}
