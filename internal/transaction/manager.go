package transaction

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TransactionStatus represents the status of a transaction
type TransactionStatus string

const (
	StatusPending    TransactionStatus = "pending"
	StatusCommitted  TransactionStatus = "committed"
	StatusRolledback TransactionStatus = "rolledback"
)

// OperationType represents the type of file operation
type OperationType string

const (
	OpMove   OperationType = "move"
	OpRename OperationType = "rename"
	OpDelete OperationType = "delete"
	OpMkdir  OperationType = "mkdir"
)

// ExecutedOperation represents a single file operation that was executed
type ExecutedOperation struct {
	Type   OperationType `json:"type"`
	Source string        `json:"source"`
	Target string        `json:"target"`
	Backup string        `json:"backup"` // Backup path for rollback
}

// Transaction represents a transaction with multiple operations
type Transaction struct {
	ID         string                `json:"id"`
	Timestamp  time.Time             `json:"timestamp"`
	Operations []*ExecutedOperation  `json:"operations"`
	Status     TransactionStatus     `json:"status"`
}

// Manager handles transaction logging and rollback
type Manager struct {
	logPath string
	mu      sync.Mutex
	txns    map[string]*Transaction
}

// NewManager creates a new transaction manager
func NewManager(logPath string) *Manager {
	return &Manager{
		logPath: logPath,
		txns:    make(map[string]*Transaction),
	}
}

// Begin starts a new transaction
func (m *Manager) Begin() *Transaction {
	m.mu.Lock()
	defer m.mu.Unlock()

	tx := &Transaction{
		ID:         generateTransactionID(),
		Timestamp:  time.Now(),
		Operations: make([]*ExecutedOperation, 0),
		Status:     StatusPending,
	}

	m.txns[tx.ID] = tx
	return tx
}

// Commit commits a transaction and persists it to the log
func (m *Manager) Commit(tx *Transaction) error {
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Update transaction status
	tx.Status = StatusCommitted

	// Persist to log file
	if err := m.persistTransaction(tx); err != nil {
		return fmt.Errorf("failed to persist transaction: %w", err)
	}

	return nil
}

// Rollback rolls back a transaction by reversing its operations
func (m *Manager) Rollback(tx *Transaction) error {
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Reverse operations in reverse order
	for i := len(tx.Operations) - 1; i >= 0; i-- {
		op := tx.Operations[i]

		switch op.Type {
		case OpMove, OpRename:
			// Restore from backup
			if op.Backup != "" {
				if err := os.Rename(op.Target, op.Source); err != nil {
					return fmt.Errorf("failed to rollback operation: %w", err)
				}
			}
		case OpDelete:
			// Restore from backup
			if op.Backup != "" {
				if err := os.Rename(op.Backup, op.Source); err != nil {
					return fmt.Errorf("failed to rollback delete operation: %w", err)
				}
			}
		case OpMkdir:
			// Remove created directory if empty
			if err := os.Remove(op.Target); err != nil {
				// Ignore error if directory is not empty
			}
		}
	}

	tx.Status = StatusRolledback

	// Persist rollback status
	if err := m.persistTransaction(tx); err != nil {
		return fmt.Errorf("failed to persist rollback: %w", err)
	}

	return nil
}

// GetHistory retrieves transaction history from the log file
func (m *Manager) GetHistory(limit int) ([]*Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load transactions from file
	txns, err := m.loadTransactionsFromFile()
	if err != nil {
		return nil, fmt.Errorf("failed to load transaction history: %w", err)
	}

	// Apply limit
	if limit > 0 && len(txns) > limit {
		txns = txns[len(txns)-limit:]
	}

	return txns, nil
}

// Undo reverses the last committed transaction
func (m *Manager) Undo(transactionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load transaction from file
	tx, err := m.loadTransactionFromFile(transactionID)
	if err != nil {
		return fmt.Errorf("failed to load transaction: %w", err)
	}

	if tx == nil {
		return fmt.Errorf("transaction not found: %s", transactionID)
	}

	if tx.Status != StatusCommitted {
		return fmt.Errorf("cannot undo transaction with status: %s", tx.Status)
	}

	// Reverse operations in reverse order
	for i := len(tx.Operations) - 1; i >= 0; i-- {
		op := tx.Operations[i]

		switch op.Type {
		case OpMove, OpRename:
			// Restore from backup
			if op.Backup != "" {
				if err := os.Rename(op.Target, op.Source); err != nil {
					return fmt.Errorf("failed to undo operation: %w", err)
				}
			}
		case OpDelete:
			// Restore from backup
			if op.Backup != "" {
				if err := os.Rename(op.Backup, op.Source); err != nil {
					return fmt.Errorf("failed to undo delete operation: %w", err)
				}
			}
		case OpMkdir:
			// Remove created directory if empty
			if err := os.Remove(op.Target); err != nil {
				// Ignore error if directory is not empty
			}
		}
	}

	tx.Status = StatusRolledback

	// Persist undo status
	if err := m.persistTransaction(tx); err != nil {
		return fmt.Errorf("failed to persist undo: %w", err)
	}

	return nil
}

// AddOperation adds an operation to a transaction
func (m *Manager) AddOperation(tx *Transaction, op *ExecutedOperation) {
	if tx != nil && op != nil {
		tx.Operations = append(tx.Operations, op)
	}
}

// persistTransaction writes a transaction to the log file
func (m *Manager) persistTransaction(tx *Transaction) error {
	// Ensure log directory exists
	logDir := filepath.Dir(m.logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Load existing transactions
	txns, _ := m.loadTransactionsFromFile()

	// Add or update transaction
	found := false
	for i, t := range txns {
		if t.ID == tx.ID {
			txns[i] = tx
			found = true
			break
		}
	}
	if !found {
		txns = append(txns, tx)
	}

	// Write to file
	data, err := json.MarshalIndent(txns, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal transactions: %w", err)
	}

	if err := os.WriteFile(m.logPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write log file: %w", err)
	}

	return nil
}

// loadTransactionsFromFile loads all transactions from the log file
func (m *Manager) loadTransactionsFromFile() ([]*Transaction, error) {
	// If file doesn't exist, return empty list
	if _, err := os.Stat(m.logPath); os.IsNotExist(err) {
		return []*Transaction{}, nil
	}

	data, err := os.ReadFile(m.logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	var txns []*Transaction
	if err := json.Unmarshal(data, &txns); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transactions: %w", err)
	}

	return txns, nil
}

// loadTransactionFromFile loads a specific transaction from the log file
func (m *Manager) loadTransactionFromFile(transactionID string) (*Transaction, error) {
	txns, err := m.loadTransactionsFromFile()
	if err != nil {
		return nil, err
	}

	for _, tx := range txns {
		if tx.ID == transactionID {
			return tx, nil
		}
	}

	return nil, nil
}

// generateTransactionID generates a unique transaction ID
func generateTransactionID() string {
	return fmt.Sprintf("txn_%d", time.Now().UnixNano())
}
