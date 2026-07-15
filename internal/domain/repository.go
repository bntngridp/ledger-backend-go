package domain

import "github.com/google/uuid"

type UserRepository interface {
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id uuid.UUID) (*User, error)
	GetUserByGoogleID(googleID string) (*User, error)
	CheckEmailExists(email string) (bool, error)
	CheckUsernameExists(username string) (bool, error)
	CreateUserWithWallet(user *User, wallet *Wallet) error
	UpdateUser(user *User) error
}

type WalletRepository interface {
	GetWalletByUserID(userID uuid.UUID) (*Wallet, error)
}

type TransactionRepository interface {
	ExecuteTransferTx(sender, recipient *Wallet, amount int64, notes string) error
	ExecuteTopUpTx(wallet *Wallet, amount int64, notes string) (*Transaction, int64, error)
	GetTransactionsByWalletID(walletID uuid.UUID) ([]Transaction, error)
}
