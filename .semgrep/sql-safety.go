package fixtures

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

func unsafeSQL(db *DB, input string) {
	// ruleid: assetforge.sql-query-construction
	db.Raw("SELECT * FROM users WHERE email = '" + input + "'")
	query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", input)
	// ruleid: assetforge.sql-query-construction
	db.Exec(query)
	// ruleid: assetforge.sql-query-construction
	db.Query(fmt.Sprintf("SELECT * FROM users WHERE id = %s", input))
	query = "DELETE FROM users WHERE email = '" + input + "'"
	// ruleid: assetforge.sql-query-construction
	db.ExecContext(nil, query)
}

func safeSQL(db *DB, input string) {
	// ok: assetforge.sql-query-construction
	db.Raw("SELECT * FROM users WHERE email = ?", input)
	// ok: assetforge.sql-query-construction
	db.Exec("DELETE FROM users WHERE email = ?", "prefix"+input)
	// ok: assetforge.sql-query-construction
	db.Query("SELECT * FROM users WHERE id = $1", input)
	const query = "SELECT * FROM users"
	// ok: assetforge.sql-query-construction
	db.Raw(query)
}

func requestSQL(db *DB, request *gin.Context) {
	query := request.Query("sql")
	// ruleid: assetforge.sql-query-construction
	db.Raw(query)
	// ok: assetforge.sql-query-construction
	db.Raw("SELECT * FROM users WHERE email = ?", request.Query("email"))
}

type DB struct{}

func (*DB) Raw(string, ...interface{})                      {}
func (*DB) Exec(string, ...interface{})                     {}
func (*DB) Query(string, ...interface{})                    {}
func (*DB) ExecContext(interface{}, string, ...interface{}) {}
