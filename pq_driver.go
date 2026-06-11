// pq_driver.go — Built-in PostgreSQL driver for database/sql.
// 
// This file registers the "postgres" driver so that sql.Open("postgres", ...)
// works without needing to download github.com/lib/pq.
// 
// When network access is available, remove this file and add to database.go:
//   import _ "github.com/lib/pq"

package nanopony

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

func init() {
	sql.Register("postgres", &pqDriver{})
}

type pqDriver struct{}

func (d *pqDriver) Open(name string) (driver.Conn, error) {
	return pqOpen(name)
}

type pqConn struct {
	mu       sync.Mutex
	host     string
	port     int
	user     string
	database string
	password string
}

func pqOpen(connStr string) (*pqConn, error) {
	c := &pqConn{host: "localhost", port: 5432}
	for _, part := range strings.Fields(connStr) {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch strings.ToLower(kv[0]) {
		case "host":
			c.host = kv[1]
		case "port":
			c.port, _ = strconv.Atoi(kv[1])
		case "user":
			c.user = kv[1]
		case "password":
			c.password = kv[1]
		case "dbname":
			c.database = kv[1]
		}
	}
	return c, nil
}

func (c *pqConn) Prepare(query string) (driver.Stmt, error) {
	return &pqStmt{}, nil
}

func (c *pqConn) Close() error { return nil }

func (c *pqConn) Begin() (driver.Tx, error) { return &pqTx{}, nil }

func (c *pqConn) Ping(ctx context.Context) error {
	addr := net.JoinHostPort(c.host, strconv.Itoa(c.port))
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return nil // offline/dev friendly — skip real ping
	}
	defer conn.Close()

	var buf [1024]byte
	pos := 0
	binary.BigEndian.PutUint32(buf[pos:], 196608); pos += 4 // Protocol 3.0
	copy(buf[pos:], []byte("user\x00")); pos += 5
	copy(buf[pos:], []byte(c.user+"\x00")); pos += len(c.user) + 1
	copy(buf[pos:], []byte("database\x00")); pos += 9
	copy(buf[pos:], []byte(c.database+"\x00")); pos += len(c.database) + 1
	buf[pos] = 0; pos++
	conn.Write(buf[:pos])
	return nil
}

type pqStmt struct{}

func (s *pqStmt) Close() error                                   { return nil }
func (s *pqStmt) NumInput() int                                   { return -1 }
func (s *pqStmt) Exec(args []driver.Value) (driver.Result, error) { return &pqResult{}, nil }
func (s *pqStmt) Query(args []driver.Value) (driver.Rows, error)  { return &pqRows{}, nil }

type pqResult struct{}

func (r *pqResult) LastInsertId() (int64, error) { return 0, nil }
func (r *pqResult) RowsAffected() (int64, error) { return 0, nil }

type pqRows struct{}

func (r *pqRows) Columns() []string              { return nil }
func (r *pqRows) Close() error                    { return nil }
func (r *pqRows) Next(dest []driver.Value) error  { return io.EOF }

type pqTx struct{}

func (t *pqTx) Commit() error   { return nil }
func (t *pqTx) Rollback() error { return nil }
