package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

type APIServer struct {
	listenAddr string
	store      Storage
}

func NewAPIServer(listenAddr string, store Storage) *APIServer {
	return &APIServer{ // return pointer to APIServer
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *APIServer) Run() { // use github.com/gorilla/mux (stable package, so i use it)
	router := mux.NewRouter()

	router.HandleFunc("/login", makeHTTPHandleFunc(s.handleLogin)) // use acc no. and pw to login, outputs the JWT token

	router.HandleFunc("/account", makeHTTPHandleFunc(s.handleAccount)) // leave open for dev purposes (later wrap this up with some auth)

	router.HandleFunc("/account/{id}", withJWTAuth(makeHTTPHandleFunc(s.handleGetAccountByID), s.store)) // wrap this with withJWTAuth to protect this
	router.HandleFunc("/account/{id}/transfer", withJWTAuth(makeHTTPHandleFunc(s.handleTransfer), s.store))
	router.HandleFunc("/account/{id}/update", withJWTAuth(makeHTTPHandleFunc(s.handleUpdateAccount), s.store))

	log.Printf("JSON API server running on%s", s.listenAddr)
	http.ListenAndServe(s.listenAddr, router)
}

// handling
// 4993187
func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		return fmt.Errorf("method not allowed: %s", r.Method)
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	acc, err := s.store.GetAccountByNumber(int(req.Number))
	if err != nil {
		return err
	}

	if !acc.ValidPassword(req.Password) {
		return fmt.Errorf("login failed")
	}

	token, err := createJWT(acc)
	if err != nil {
		return err
	}

	res := LoginResponse{
		Token:  token,
		Number: acc.Number,
	}

	return WriteJSON(w, http.StatusOK, res)
}

// convert function to http handler to avoid clash with mux handleFunc
func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return s.handleGetAccount(w, r)
	}
	if r.Method == "POST" {
		return s.handleCreateAccount(w, r)
	}

	return fmt.Errorf("method not allowed: %s", r.Method)
}

// GET /account, hence we name handleGetAccount instead of accounts
func (s *APIServer) handleGetAccount(w http.ResponseWriter, r *http.Request) error {
	accounts, err := s.store.GetAccounts()
	if err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, accounts)
}

func (s *APIServer) handleGetAccountByID(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		id, err := getID(r)
		if err != nil {
			return err
		}

		account, err := s.store.GetAccountByID(id)
		if err != nil {
			return err
		}

		// db.get(id)
		return WriteJSON(w, http.StatusOK, account)
	}

	if r.Method == "DELETE" {
		return s.handleDeleteAccount(w, r)
	}

	if r.Method == "PUT" {
		return s.handleUpdateAccount(w, r)
	}

	return fmt.Errorf("method not allowed %s", r.Method)
}

func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {
	createAccountReq := new(CreateAccountRequest)

	if err := json.NewDecoder(r.Body).Decode(createAccountReq); err != nil {
		return err
	}
	account, err := NewAccount(createAccountReq.FirstName, createAccountReq.LastName, createAccountReq.Password) // can also do Account{}
	if err != nil {
		return err
	}

	if err := s.store.CreateAccount(account); err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, account)
}

func (s *APIServer) handleUpdateAccount(w http.ResponseWriter, r *http.Request) error {
	id, err := getID(r)
	if err != nil {
		return err
	}

	oldAccount, err := s.store.GetAccountByID(id)
	if err != nil {
		return err
	}

	var newAccount Account
	if err := json.NewDecoder(r.Body).Decode(&newAccount); err != nil {
		return err
	}

	updatedAccount := &Account{ // set to pointer to this account to abide by s.store.UpdateAccount() params
		ID:        oldAccount.ID,
		Number:    oldAccount.Number,
		FirstName: newAccount.FirstName,
		LastName:  newAccount.LastName,
		Balance:   newAccount.Balance,
		CreatedAt: oldAccount.CreatedAt,
	}

	if err := s.store.UpdateAccount(updatedAccount); err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, updatedAccount)
}

func (s *APIServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request) error {
	id, err := getID(r)
	if err != nil {
		return err
	}

	if err := s.store.DeleteAccount(id); err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, map[string]int{"deleted": id})

}

func (s *APIServer) handleTransfer(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		return fmt.Errorf("method not allowed: %s", r.Method)
	}

	fromId, err := getID(r)
	if err != nil {
		return err
	}

	toNumber, amount, err := decodeTransfer(r)
	if err != nil {
		return err
	}

	toAccount, err := s.store.GetAccountByNumber(toNumber)
	if err != nil {
		return err
	}

	fromAccount, err := s.store.GetAccountByID(fromId)
	if err != nil {
		return err
	}

	// transfer the money
	acc, err := s.store.TransferToAccount(toAccount, fromAccount, amount)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}
	// show the new balance for the owners account (not the other persons acc)
	transferResponse := TransferResponse{
		Number:  acc.Number,
		Balance: acc.Balance,
	}
	fmt.Printf("Transferring: $%d to %s %s\n", amount, toAccount.FirstName, toAccount.LastName)

	return WriteJSON(w, http.StatusOK, transferResponse)
}

// Helper functions
func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json") // write json correctly
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(v)
}

// Delete for production
// Windows
// 		SET JWT_SECRET=noah1111

// Linux
// 		export JWT_SECRET=noah1111

func createJWT(account *Account) (string, error) {
	claims := &jwt.MapClaims{
		"expiresAt":     jwt.NewNumericDate(time.Unix(1516239022, 0)),
		"accountNumber": account.Number,
	}

	secret := os.Getenv("JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))

}

func permissionDenied(w http.ResponseWriter) {
	WriteJSON(w, http.StatusForbidden, APIError{Error: "permission denied"})
}

func withJWTAuth(handlerFunc http.HandlerFunc, s Storage) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Calling JWT auth middleware")

		tokenString := r.Header.Get("x-jwt-token")

		token, err := validateJWT(tokenString)

		if err != nil {
			permissionDenied(w)
			return
		}

		if !token.Valid {
			permissionDenied(w)
			return
		}

		userID, err := getID(r)
		if err != nil {
			permissionDenied(w)
			return
		}

		account, err := s.GetAccountByID(userID)
		if err != nil {
			permissionDenied(w)
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		if account.Number != int64(claims["accountNumber"].(float64)) { // account.Number is int64 and claims["accountNumber"] is float64
			permissionDenied(w)
			return
		}

		if err != nil {
			WriteJSON(w, http.StatusForbidden, APIError{Error: "invalid token"})
			return
		}

		handlerFunc(w, r)
	}
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_SECRET")
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {

		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(secret), nil
	})
}

type APIFunc func(http.ResponseWriter, *http.Request) error // func signature of func we're using

type APIError struct {
	Error string `json:"error"`
}

func makeHTTPHandleFunc(f APIFunc) http.HandlerFunc { // decorate API func and to http handler func
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
		}
	}
}

func getID(r *http.Request) (int, error) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return id, fmt.Errorf("invalid id given: %s", idStr) // return a user friendly error
	}

	return id, nil
}

func decodeTransfer(r *http.Request) (int, int, error) {
	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return -1, -1, err
	}

	return req.ToAccountNumber, req.Amount, nil
}
