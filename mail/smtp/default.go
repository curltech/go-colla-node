package smtp

import (
	"crypto/tls"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/emersion/go-sasl"
	"time"

	"github.com/emersion/go-smtp"
)

func Start() {
	if !config.SmtpServerParams.Enable {
		logger.Errorf("smtp is not enable")
		return
	}

	be := &Backend{}

	smtpServer := smtp.NewServer(be)

	smtpServer.Addr = config.SmtpServerParams.Addr
	smtpServer.Domain = config.SmtpServerParams.Domain
	smtpServer.ReadTimeout = time.Duration(config.SmtpServerParams.ReadTimeout)
	smtpServer.WriteTimeout = time.Duration(config.SmtpServerParams.WriteTimeout)
	smtpServer.MaxMessageBytes = int(config.SmtpServerParams.MaxMessageBytes)
	smtpServer.MaxRecipients = config.SmtpServerParams.MaxRecipients

	// Add deprecated LOGIN auth method as some clients haven't learned
	smtpServer.EnableAuth(sasl.Login, func(conn *smtp.Conn) sasl.Server {
		return sasl.NewLoginServer(func(username, password string) error {
			state := conn.State()
			session, err := be.Login(&state, username, password)
			if err != nil {
				return err
			}

			conn.SetSession(session)
			return nil
		})
	})

	// force TLS for auth
	smtpServer.AllowInsecureAuth = false
	// Load the certificate and key
	cer, err := tls.LoadX509KeyPair(config.TlsParams.Cert, config.TlsParams.Key)
	if err != nil {
		logger.Errorf(err.Error())
		return
	}
	// Configure the TLS support
	smtpServer.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cer}}

	logger.Infof("Starting server at", smtpServer.Addr)
	if err := smtpServer.ListenAndServe(); err != nil {
		logger.Errorf(err.Error())
	}
}
