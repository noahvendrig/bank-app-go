package main

import "math/rand"

type Account struct {
	ID        int    `json:"id"` // instructions for serialising into JSON
	FirstName string `json:"FirstName"`
	LastName  string `json:"LastName"`
	Number    int64  `json:"Number"`
	Balance   int64  `json:"Balance"`
}

func NewAccount(firstName, lastName string) *Account { // return pointer to Account
	return &Account{
		ID:        rand.Intn(10000),
		FirstName: firstName,
		LastName:  lastName,
		Number:    int64(rand.Intn(10000000)),
	}
}
