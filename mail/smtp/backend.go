package smtp

import (
	"errors"
	"github.com/curltech/go-colla-node/mail/entity"
	"github.com/curltech/go-colla-node/mail/service"
	"github.com/emersion/go-smtp"
)

// The Backend implements SMTP server methods.
type Backend struct{}

// Login handles a login command with username and password.
func (bkd *Backend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	account := &entity.MailAccount{}
	account.Name = username
	account.Password = password
	found := service.GetMailAccountService().Get(account, false, "", "")
	if !found {
		return nil, errors.New("Bad name or password")
	}

	return &Session{}, nil
}

// AnonymousLogin requires clients to authenticate using SMTP AUTH before sending emails
func (bkd *Backend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return nil, smtp.ErrAuthRequired
}
