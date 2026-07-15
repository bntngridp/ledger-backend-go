package domain

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

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
	GetWalletBalance(walletID uuid.UUID, assetSymbol string) (*WalletBalance, error)
	GetBalancesByWalletID(walletID uuid.UUID) ([]WalletBalance, error)
}

type TransactionRepository interface {
	ExecuteTransferTx(senderWalletID, recipientWalletID uuid.UUID, amount decimal.Decimal, assetSymbol string, notes string) error
	ExecuteTopUpTx(walletID uuid.UUID, amount decimal.Decimal, assetSymbol string, notes string) (*Transaction, decimal.Decimal, error)
	GetTransactionsByWalletID(walletID uuid.UUID) ([]Transaction, error)
}

type CryptoAddressRepository interface {
	GetAddressByWalletID(walletID uuid.UUID, network, assetSymbol string) (*CryptoAddress, error)
	GetAddressByValue(address string) (*CryptoAddress, error)
	CreateAddress(cryptoAddr *CryptoAddress) error
}
