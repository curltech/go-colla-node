package imap

import (
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/mail/imap/backend"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap-idle"
	"github.com/emersion/go-imap/server"
	"github.com/emersion/go-sasl"
)

func Start() {
	if !config.ImapServerParams.Enable {
		logger.Errorf("imap is not enable")
		return
	}
	// Create a memory backend
	be := backend.GetBackend()
	// Create a new server
	imapServer := server.New(be)
	imapServer.Addr = config.ImapServerParams.Addr
	imapServer.EnableAuth(sasl.Login, func(conn server.Conn) sasl.Server {
		return sasl.NewLoginServer(func(username, password string) error {
			user, err := conn.Server().Backend.Login(nil, username, password)
			if err != nil {
				return err
			}

			ctx := conn.Context()
			ctx.State = imap.AuthenticatedState
			ctx.User = user
			return nil
		})
	})
	imapServer.Enable(idle.NewExtension())

	logger.Infof("Starting IMAP server at localhost:1143")
	if err := imapServer.ListenAndServe(); err != nil {
		logger.Errorf(err.Error())
	}
}
