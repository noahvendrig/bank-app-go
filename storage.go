package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// docker run --name bank-pg -e POSTGRES_PASSWORD=bankappgo -p 5432:5432 -d postgres
// docker stop bank-pg
// docker remove bank-pg

type Storage interface {
	CreateAccount(*Account) error
	DeleteAccount(int) error
	UpdateAccount(*Account) error
	GetAccounts() ([]*Account, error)
	GetAccountByNumber(int) (*Account, error)
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
			first_name varchar(100),
			last_name varchar(100),
			number serial,
			encrypted_password varchar(100),
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
		(first_name, last_name, number, encrypted_password, balance, created_at)
		VALUES
		($1, $2, $3, $4, $5, $6)
	` // $N is index of arguments passed to db.Exec() - prevent hardcoding values in SQL)

	_, err := s.db.Query(
		query,
		acc.FirstName,
		acc.LastName,
		acc.Number,
		acc.EncryptedPassword,
		acc.Balance,
		acc.CreatedAt,
	)

	if err != nil {
		return err
	}
	// fmt.Printf("%+v\n", r)

	return nil
}

func (s *PostgresStore) UpdateAccount(acc *Account) error {
	query := `
		UPDATE Account
		SET first_name=$2, last_name=$3, number=$4, encrypted_password=$5, balance=$6, created_at=$7
		WHERE id=$1
	`

	_, err := s.db.Query(
		query,
		acc.ID,
		acc.FirstName,
		acc.LastName,
		acc.Number,
		acc.EncryptedPassword,
		acc.Balance,
		acc.CreatedAt,
	)

	if err != nil {
		return err
	}
	// fmt.Printf("%+v\n", r)

	return nil
}

func (s *PostgresStore) DeleteAccount(id int) error {
	_, err := s.db.Query("DELETE FROM Account WHERE id=$1", id) // hard delete

	return err
}

func (s *PostgresStore) GetAccountByNumber(number int) (*Account, error) {
	rows, err := s.db.Query("SELECT * FROM Account WHERE number=$1", number)

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		return scanIntoAccount(rows)
	}

	return nil, fmt.Errorf("account with number [%d] not found", number)
}

func (s *PostgresStore) GetAccountByID(id int) (*Account, error) {
	rows, err := s.db.Query("SELECT * FROM Account WHERE id=$1", id)

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		return scanIntoAccount(rows)
	}

	return nil, fmt.Errorf("account with id [%d] not found", id)
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
		&account.EncryptedPassword,
		&account.Balance,
		&account.CreatedAt,
	)

	return account, err
}
