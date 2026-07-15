package blockchain

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"strings"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/shopspring/decimal"

	"github.com/bntngridp/ledger-backend/internal/domain"
)

const (
	minConfirmations = 3
	maxBackoffSec    = 60
)

type ListenerDeps struct {
	AlchemyClient     *AlchemyClient
	CryptoAddressRepo domain.CryptoAddressRepository
	TransactionRepo   domain.TransactionRepository
	ContractAssets    map[string]string
	ContractDecimals  map[string]int
	Network           string
}

type ERC20Listener struct {
	deps      ListenerDeps
	watchList map[string]*domain.CryptoAddress
	watchMu   sync.RWMutex
}

func NewERC20Listener(deps ListenerDeps) *ERC20Listener {
	return &ERC20Listener{
		deps:      deps,
		watchList: make(map[string]*domain.CryptoAddress),
	}
}

// Start spawns the listener loop, utilizing exponential backoff on reconnection failures.
func (l *ERC20Listener) Start(ctx context.Context) {
	slog.Info("[ERC20Listener] Starting", "network", l.deps.Network)
	backoff := 1

	for {
		select {
		case <-ctx.Done():
			slog.Info("[ERC20Listener] Context cancelled, shutting down")
			return
		default:
		}

		if err := l.refreshWatchList(ctx); err != nil {
			slog.Error("[ERC20Listener] Failed to refresh watch list", "error", err)
		}

		if err := l.listen(ctx); err != nil {
			slog.Error("[ERC20Listener] Listener error", "error", err, "reconnect_delay_seconds", backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(backoff) * time.Second):
			}
			backoff = int(math.Min(float64(backoff*2), float64(maxBackoffSec)))
		} else {
			backoff = 1
		}
	}
}

func (l *ERC20Listener) refreshWatchList(ctx context.Context) error {
	addresses, err := l.deps.CryptoAddressRepo.GetAllAddresses(l.deps.Network)
	if err != nil {
		return fmt.Errorf("failed to load watch list: %w", err)
	}

	l.watchMu.Lock()
	defer l.watchMu.Unlock()
	for i := range addresses {
		addr := &addresses[i]
		l.watchList[strings.ToLower(addr.Address)] = addr
	}
	slog.Info("[ERC20Listener] Watch list loaded", "count", len(l.watchList))
	return nil
}

func (l *ERC20Listener) AddToWatchList(addr *domain.CryptoAddress) {
	l.watchMu.Lock()
	defer l.watchMu.Unlock()
	l.watchList[strings.ToLower(addr.Address)] = addr
}

func (l *ERC20Listener) listen(ctx context.Context) error {
	contractAddresses := make([]common.Address, 0, len(l.deps.ContractAssets))
	for addrStr := range l.deps.ContractAssets {
		contractAddresses = append(contractAddresses, common.HexToAddress(addrStr))
	}

	query := ethereum.FilterQuery{
		Addresses: contractAddresses,
	}

	wsClient, sub, logsChan, err := l.deps.AlchemyClient.SubscribeToLogs(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to subscribe to logs: %w", err)
	}
	defer wsClient.Close()

	slog.Info("[ERC20Listener] Subscribed to contracts. Listening...", "count", len(contractAddresses))

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-sub.Err():
			return fmt.Errorf("subscription error: %w", err)
		case vLog := <-logsChan:
			l.handleLog(ctx, vLog)
		}
	}
}

func (l *ERC20Listener) handleLog(ctx context.Context, vLog types.Log) {
	// Skip removed logs (chain reorganization)
	if vLog.Removed {
		return
	}

	event, err := DecodeTransferEvent(vLog.Data, vLog.Topics)
	if err != nil || event == nil {
		return
	}

	toAddrLower := strings.ToLower(event.To.Hex())
	l.watchMu.RLock()
	cryptoAddr, found := l.watchList[toAddrLower]
	l.watchMu.RUnlock()
	if !found {
		return
	}

	contractLower := strings.ToLower(vLog.Address.Hex())
	assetSymbol, ok := l.deps.ContractAssets[contractLower]
	if !ok {
		return
	}

	decimals, ok := l.deps.ContractDecimals[contractLower]
	if !ok {
		decimals = 18
	}

	txHash := vLog.TxHash.Hex()

	if err := l.waitForConfirmations(ctx, vLog.BlockNumber); err != nil {
		slog.Error("[ERC20Listener] Confirmation check failed", "tx", txHash, "error", err)
		return
	}

	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	amountDecimal := decimal.NewFromBigInt(event.Value, 0).Div(decimal.NewFromBigInt(divisor, 0))

	notes := fmt.Sprintf("On-chain deposit on %s from %s", l.deps.Network, event.From.Hex())
	_, depositErr := l.deps.TransactionRepo.CreditCryptoDeposit(
		cryptoAddr.WalletID,
		amountDecimal,
		assetSymbol,
		txHash,
		notes,
	)

	if depositErr != nil {
		if depositErr == domain.ErrDuplicateTransaction {
			slog.Warn("[ERC20Listener] Duplicate transaction hash, skipping", "tx", txHash)
		} else {
			slog.Error("[ERC20Listener] Failed to credit deposit", "tx", txHash, "error", depositErr)
		}
		return
	}

	slog.Info("[ERC20Listener] Deposit credited successfully",
		"amount", amountDecimal.String(),
		"asset", assetSymbol,
		"wallet_id", cryptoAddr.WalletID.String(),
		"tx", txHash,
	)
}

// waitForConfirmations polls until the current block is at least minConfirmations ahead of txBlock.
func (l *ERC20Listener) waitForConfirmations(ctx context.Context, txBlock uint64) error {
	target := txBlock + minConfirmations
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			current, err := l.deps.AlchemyClient.GetCurrentBlock(ctx)
			if err != nil {
				slog.Error("[ERC20Listener] Block number check failed", "error", err)
				continue
			}
			if current >= target {
				return nil
			}
			slog.Info("[ERC20Listener] Waiting for confirmations", "current_block", current, "target_block", target)
		}
	}
}
