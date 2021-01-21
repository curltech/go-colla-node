package smtp

import (
	"crypto/tls"
	"github.com/emersion/go-sasl"
	"log"
	"time"

	"github.com/emersion/go-smtp"
)

func main() {
	be := &Backend{}

	s := smtp.NewServer(be)

	s.Addr = ":1025"
	s.Domain = "localhost"
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = 1024 * 1024
	s.MaxRecipients = 50

	// Add deprecated LOGIN auth method as some clients haven't learned
	s.EnableAuth(sasl.Login, func(conn *smtp.Conn) sasl.Server {
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
	s.AllowInsecureAuth = false
	// Load the certificate and key
	cer, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatal(err)
		return
	}
	// Configure the TLS support
	s.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cer}}

	log.Println("Starting server at", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
