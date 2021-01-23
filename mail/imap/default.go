package imap

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/mail/imap/dbbackend"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-sasl"
	"log"

	"github.com/emersion/go-imap-idle"
	"github.com/emersion/go-imap/server"
)

func Start() {
	// Create a memory backend
	be := backend.GetBackend()

	// Create a new server
	s := server.New(be)
	s.Addr = ":1143"
	s.EnableAuth(sasl.Login, func(conn server.Conn) sasl.Server {
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
	s.Enable(idle.NewExtension())

	log.Println("Starting IMAP server at localhost:1143")
	if err := s.ListenAndServe(); err != nil {
		logger.Errorf(err.Error())
	}
}
