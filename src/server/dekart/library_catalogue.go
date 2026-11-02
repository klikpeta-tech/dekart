package dekart

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"dekart/src/proto"
	"dekart/src/server/conn"
	"dekart/src/server/library"
	"dekart/src/server/secrets"
)

// EnableLibraryCatalogue wires the catalogue reader to the library database through the library connection
// (DEKART_LIBRARY_CONNECTION_ID). It does nothing when no library connection is configured, so the library
// commands keep answering Unavailable. The database is opened on first use; a rotated password needs a restart.
func (s *Server) EnableLibraryCatalogue() {
	if library.ConnectionID() == "" {
		return
	}
	s.SetLibraryCatalogue(library.NewPGCatalogue(s.openLibraryDB))
}

// openLibraryDB opens the library database with the credentials stored on the library connection.
func (s *Server) openLibraryDB(ctx context.Context) (*sql.DB, error) {
	connection, err := s.getConnection(ctx, library.ConnectionID())
	if err != nil {
		return nil, err
	}
	if connection == nil || connection.ConnectionType != proto.ConnectionType_CONNECTION_TYPE_POSTGRES {
		return nil, fmt.Errorf("the library connection is missing or is not a Postgres connection")
	}
	// The password is encrypted by the server, so no user claims are needed to read it.
	password := secrets.SecretToString(connection.PostgresPassword, nil)
	if password == "" {
		return nil, fmt.Errorf("the library connection has no password")
	}
	dsn, err := conn.BuildPostgresKeywordDSN(connection, password)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	// The reader role allows few sessions and library runs use it too; idle ones go before Neon suspends them.
	db.SetMaxOpenConns(2)
	db.SetConnMaxIdleTime(4 * time.Minute)
	return db, nil
}
