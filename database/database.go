package database

import (
	"encoding/json"
	"fmt"
	"time"

	badger "github.com/dgraph-io/badger/v4"
)

type DB struct {
	*badger.DB
}

type Transaction struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Type        string    `json:"type"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type User struct {
	UserID       int64     `json:"user_id"`
	Username     string    `json:"username"`
	RegisteredAt time.Time `json:"registered_at"`
}

func NewDB(path string) (*DB, error) {
	opts := badger.DefaultOptions(path)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

func (db *DB) RegisterUser(userID int64, username string) error {
	user := User{
		UserID:       userID,
		Username:     username,
		RegisteredAt: time.Now(),
	}

	return db.Update(func(txn *badger.Txn) error {
		key := fmt.Sprintf("user:%d", userID)
		value, err := json.Marshal(user)
		if err != nil {
			return err
		}
		return txn.Set([]byte(key), value)
	})
}

func (db *DB) AddTransaction(userID int64, transType string, amount float64, description string) error {
	return db.Update(func(txn *badger.Txn) error {
		// Get the last transaction ID
		var lastID int64
		err := db.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte("last_transaction_id"))
			if err == badger.ErrKeyNotFound {
				return nil
			}
			if err != nil {
				return err
			}
			return item.Value(func(val []byte) error {
				lastID = int64(bytesToUint64(val))
				return nil
			})
		})
		if err != nil {
			return err
		}

		// Increment the ID
		lastID++

		// Create new transaction
		trans := Transaction{
			ID:          lastID,
			UserID:      userID,
			Type:        transType,
			Amount:      amount,
			Description: description,
			CreatedAt:   time.Now(),
		}

		// Save the transaction
		value, err := json.Marshal(trans)
		if err != nil {
			return err
		}

		// Save the transaction
		key := fmt.Sprintf("transaction:%d", lastID)
		if err := txn.Set([]byte(key), value); err != nil {
			return err
		}

		// Update the last transaction ID
		return txn.Set([]byte("last_transaction_id"), uint64ToBytes(uint64(lastID)))
	})
}

func (db *DB) GetBalance() (float64, error) {
	var balance float64

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("transaction:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				var trans Transaction
				if err := json.Unmarshal(v, &trans); err != nil {
					return err
				}

				if trans.Type == "income" {
					balance += trans.Amount
				} else if trans.Type == "expense" {
					balance -= trans.Amount
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	return balance, err
}

func (db *DB) GetMonthlyStats() (income float64, expenses float64, balance float64, err error) {
	startOfMonth := time.Now().UTC().Format("2006-01")

	err = db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("transaction:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				var trans Transaction
				if err := json.Unmarshal(v, &trans); err != nil {
					return err
				}

				if trans.CreatedAt.Format("2006-01") != startOfMonth {
					return nil
				}

				if trans.Type == "income" {
					income += trans.Amount
				} else if trans.Type == "expense" {
					expenses += trans.Amount
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return 0, 0, 0, err
	}

	balance = income - expenses
	return
}

func (db *DB) GetMonthlyExpenses() (float64, error) {
	startOfMonth := time.Now().UTC().Format("2006-01")
	var expenses float64

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("transaction:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				var trans Transaction
				if err := json.Unmarshal(v, &trans); err != nil {
					return err
				}

				if trans.CreatedAt.Format("2006-01") == startOfMonth && trans.Type == "expense" {
					expenses += trans.Amount
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	return expenses, err
}

func uint64ToBytes(i uint64) []byte {
	var buf [8]byte
	for shift := uint(0); shift < 64; shift += 8 {
		buf[shift/8] = byte(i >> shift)
	}
	return buf[:]
}

func bytesToUint64(b []byte) uint64 {
	var i uint64
	for shift := uint(0); shift < 64; shift += 8 {
		i |= uint64(b[shift/8]) << shift
	}
	return i
}
