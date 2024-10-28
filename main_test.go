package main_test

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	db      *sql.DB
	userID  string
	balance int64
)

func generateUsers(n int) error {
	const usersQuery = `INSERT INTO users (id, balance) VALUES ('%s', '%v') ON CONFLICT (id) DO UPDATE SET balance = EXCLUDED.balance;`

	for ; n > 0; n-- {
		userID = uuid.New().String()
		balance = rand.Int63n(1_000_000) + 1_00_000

		_, err := db.Exec(fmt.Sprintf(usersQuery, userID, balance))
		if err != nil {
			return err
		}
	}

	return nil
}

func TestMain(m *testing.M) {
	var err error

	db, err = sql.Open("pgx", "postgres://postgres:postgres@postgres:5432/postgres?sslmode=disable")
	if err != nil {
		panic(err.Error())
	}

	const createTablesQuery = `
	DROP TABLE IF EXISTS users CASCADE;	
	CREATE TABLE IF NOT EXISTS users (
		id varchar,
		balance bigint,
		PRIMARY KEY(id)
	);
	DROP TABLE IF EXISTS transactions CASCADE;
	CREATE TABLE IF NOT EXISTS transactions (
		id serial,
		amount bigint,
		user_id varchar,
		PRIMARY KEY(id),
		CONSTRAINT fk_user_id
      		FOREIGN KEY(user_id)
        		REFERENCES users(id)
	);
	`

	if _, err := db.Exec(createTablesQuery); err != nil {
		panic(err.Error())
	}

	if err := generateUsers(10000); err != nil {
		panic(err.Error())
	}

	os.Exit(m.Run())
}

func BenchmarkPerformance(b *testing.B) {
	b.StopTimer()
	var sqlQuery = fmt.Sprintf(`
	DO
	$do$
	BEGIN
		IF (SELECT balance FROM users WHERE id = '%s' FOR UPDATE) > 1
		THEN
  			INSERT INTO transactions (user_id,amount) VALUES ('%s', -1);
  			UPDATE users SET balance = balance - 1 WHERE id = '%s';
		END IF;
	END
	$do$
	`, userID, userID, userID)

	b.Log(fmt.Sprintf("user_id = %s balance = %v\n", userID, balance))

	success := int64(0)
	fail := int64(0)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		res, err := db.Exec(sqlQuery)
		if err != nil {
			b.Fatal(err.Error())
		}

		affectedRows, err := res.RowsAffected()
		if err != nil {
			b.Fatal(err.Error())
		}

		if affectedRows == 1 {
			success++
		} else if affectedRows == 0 {
			fail++
		} else {
			b.Fatal("more then 1 rows affected")
		}
	}
}
