package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// docker run--name some-postgres-e POSTGRES_PASSWORD=bankappgo-p 5432:5432 -d postgres

type Storage interface {
	CreateAccount(*Account) error
	DeleteAccount(int) error
	UpdateAccount(*Account) error
	GetAccounts() ([]*Account, error)
	GetAccountByID(int) (*Account, error)
}

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore() (*PostgresStore, error) {
	connStr := "user=postgres dbname=postgres password=bankappgo sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresStore{
		db: db,
	}, nil

}

func (s PostgresStore) Init() error {
	return s.CreateAccountTable()
}

func (s PostgresStore) CreateAccountTable() error {
	query := `
		CREATE TABLE 
		IF NOT EXISTS
		Account (
			id serial primary key,
			first_name varchar(50),
			last_name varchar(50),
			number serial,
			balance serial,
			created_at timestamp
		)`
	_, err := s.db.Exec(query)
	return err
}

// psql pw: bankappgo

func (s *PostgresStore) CreateAccount(acc *Account) error {
	query := `
	INSERT INTO ACCOUNT 
	(first_name, last_name, number, balance, created_at)
	VALUES
	($1, $2, $3, $4, $5)` // $N is index of arguments passed to db.Exec() - prevent hardcoding values in SQL)

	r, err := s.db.Query(
		query,
		acc.FirstName,
		acc.LastName,
		acc.Number,
		acc.Balance,
		acc.CreatedAt,
	)

	if err != nil {
		return err
	}
	fmt.Printf("%+v\n", r)

	return nil
}

func (s *PostgresStore) UpdateAccount(*Account) error {
	return nil
}

func (s *PostgresStore) DeleteAccount(id int) error {
	_, err := s.db.Query("DELETE FROM Account WHERE id=$1", id) // hard delete

	return err
}

func (s *PostgresStore) GetAccountByID(id int) (*Account, error) {
	rows, err := s.db.Query("SELECT * FROM Account WHERE id=$1", id)

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		return scanIntoAccount(rows)
	}

	return nil, fmt.Errorf("account %d not found", id)
}

func (s *PostgresStore) GetAccounts() ([]*Account, error) {
	rows, err := s.db.Query("SELECT * FROM Account")
	if err != nil {
		return nil, err
	}

	accounts := []*Account{}
	for rows.Next() {
		account, err := scanIntoAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func scanIntoAccount(rows *sql.Rows) (*Account, error) {
	account := new(Account)
	err := rows.Scan(
		&account.ID,
		&account.FirstName,
		&account.LastName,
		&account.Number,
		&account.Balance,
		&account.CreatedAt,
	)

	return account, err
}
