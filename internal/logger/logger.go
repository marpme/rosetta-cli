// Copyright 2020 Coinbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/reconciler"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/fatih/color"
)

const (
	// blockStreamFile contains the stream of processed
	// blocks and whether they were added or removed.
	blockStreamFile = "blocks.txt"

	// transactionStreamFile contains the stream of processed
	// transactions and whether they were added or removed.
	transactionStreamFile = "transactions.txt"

	// balanceStreamFile contains the stream of processed
	// balance changes.
	balanceStreamFile = "balance_changes.txt"

	// reconcileSuccessStreamFile contains the stream of processed
	// reconciliations.
	reconcileSuccessStreamFile = "successful_reconciliations.txt"
	reconcileFailureStreamFile = "failure_reconciliations.txt"

	// addEvent is printed in a stream
	// when an event is added.
	addEvent = "Add"

	// removeEvent is printed in a stream
	// when an event is orphaned.
	removeEvent = "Remove"

	// logFilePermissions specifies that the user can
	// read and write the file.
	logFilePermissions = 0600
)

// Logger contains all logic to record validator output
// and benchmark a Rosetta Server.
type Logger struct {
	logDir            string
	logBlocks         bool
	logTransactions   bool
	logBalanceChanges bool
	logReconciliation bool
}

// NewLogger constructs a new Logger.
func NewLogger(
	logDir string,
	logBlocks bool,
	logTransactions bool,
	logBalanceChanges bool,
	logReconciliation bool,
) *Logger {
	return &Logger{
		logDir:            logDir,
		logBlocks:         logBlocks,
		logTransactions:   logTransactions,
		logBalanceChanges: logBalanceChanges,
		logReconciliation: logReconciliation,
	}
}

// AddBlockStream writes the next processed block to the end of the
// blockStreamFile output file.
func (l *Logger) AddBlockStream(
	ctx context.Context,
	block *types.Block,
) error {
	if !l.logBlocks {
		return nil
	}

	f, err := os.OpenFile(
		path.Join(l.logDir, blockStreamFile),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		logFilePermissions,
	)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf(
		"%s Block %d:%s with Parent Block %d:%s\n",
		addEvent,
		block.BlockIdentifier.Index,
		block.BlockIdentifier.Hash,
		block.ParentBlockIdentifier.Index,
		block.ParentBlockIdentifier.Hash,
	))
	if err != nil {
		return err
	}

	return l.TransactionStream(ctx, block)
}

// RemoveBlockStream writes the next processed block to the end of the
// blockStreamFile output file.
func (l *Logger) RemoveBlockStream(
	ctx context.Context,
	block *types.BlockIdentifier,
) error {
	if !l.logBlocks {
		return nil
	}

	f, err := os.OpenFile(
		path.Join(l.logDir, blockStreamFile),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		logFilePermissions,
	)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf(
		"%s Block %d:%s\n",
		removeEvent,
		block.Index,
		block.Hash,
	))
	if err != nil {
		return err
	}

	return nil
}

// TransactionStream writes the next processed block's transactions
// to the end of the transactionStreamFile.
func (l *Logger) TransactionStream(
	ctx context.Context,
	block *types.Block,
) error {
	if !l.logTransactions {
		return nil
	}

	f, err := os.OpenFile(
		path.Join(l.logDir, transactionStreamFile),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		logFilePermissions,
	)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, tx := range block.Transactions {
		_, err = f.WriteString(fmt.Sprintf(
			"Transaction %s at Block %d:%s\n",
			tx.TransactionIdentifier.Hash,
			block.BlockIdentifier.Index,
			block.BlockIdentifier.Hash,
		))
		if err != nil {
			return err
		}

		for _, op := range tx.Operations {
			amount := ""
			symbol := ""
			if op.Amount != nil {
				amount = op.Amount.Value
				symbol = op.Amount.Currency.Symbol
			}
			participant := ""
			if op.Account != nil {
				participant = types.AccountString(op.Account)
			}

			networkIndex := op.OperationIdentifier.Index
			if op.OperationIdentifier.NetworkIndex != nil {
				networkIndex = *op.OperationIdentifier.NetworkIndex
			}

			_, err = f.WriteString(fmt.Sprintf(
				"TxOp %d(%d) %s %s %s %s %s\n",
				op.OperationIdentifier.Index,
				networkIndex,
				op.Type,
				participant,
				amount,
				symbol,
				op.Status,
			))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// BalanceStream writes a slice of storage.BalanceChanges
// to the balanceStreamFile.
func (l *Logger) BalanceStream(
	ctx context.Context,
	balanceChanges []*parser.BalanceChange,
) error {
	if !l.logBalanceChanges {
		return nil
	}

	f, err := os.OpenFile(
		path.Join(l.logDir, balanceStreamFile),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		logFilePermissions,
	)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, balanceChange := range balanceChanges {
		balanceLog := fmt.Sprintf(
			"Account: %s Change: %s:%s Block: %d:%s",
			balanceChange.Account.Address,
			balanceChange.Difference,
			types.CurrencyString(balanceChange.Currency),
			balanceChange.Block.Index,
			balanceChange.Block.Hash,
		)

		if _, err := f.WriteString(fmt.Sprintf("%s\n", balanceLog)); err != nil {
			return err
		}
	}
	return nil
}

// ReconcileSuccessStream logs all reconciliation checks performed
// during syncing.
func (l *Logger) ReconcileSuccessStream(
	ctx context.Context,
	reconciliationType string,
	account *types.AccountIdentifier,
	currency *types.Currency,
	balance string,
	block *types.BlockIdentifier,
) error {
	if !l.logReconciliation {
		return nil
	}

	f, err := os.OpenFile(
		path.Join(l.logDir, reconcileSuccessStreamFile),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		logFilePermissions,
	)
	if err != nil {
		return err
	}
	defer f.Close()

	log.Printf(
		"%s Reconciled %s at %d\n",
		reconciliationType,
		types.AccountString(account),
		block.Index,
	)

	_, err = f.WriteString(fmt.Sprintf(
		"Type:%s Account: %s Currency: %s Balance: %s Block: %d:%s\n",
		reconciliationType,
		types.AccountString(account),
		types.CurrencyString(currency),
		balance,
		block.Index,
		block.Hash,
	))
	if err != nil {
		return err
	}

	return nil
}

// ReconcileFailureStream logs all reconciliation checks performed
// during syncing.
func (l *Logger) ReconcileFailureStream(
	ctx context.Context,
	reconciliationType string,
	account *types.AccountIdentifier,
	currency *types.Currency,
	computedBalance string,
	nodeBalance string,
	block *types.BlockIdentifier,
) error {
	// Always print out reconciliation failures
	if reconciliationType == reconciler.InactiveReconciliation {
		color.Yellow(
			"Missing balance-changing operation detected for %s computed balance: %s%s node balance: %s%s",
			types.AccountString(account),
			computedBalance,
			currency.Symbol,
			nodeBalance,
			currency.Symbol,
		)
	} else {
		color.Yellow(
			"Reconciliation failed for %s at %d computed: %s%s node: %s%s",
			types.AccountString(account),
			block.Index,
			computedBalance,
			currency.Symbol,
			nodeBalance,
			currency.Symbol,
		)
	}

	if !l.logReconciliation {
		return nil
	}

	f, err := os.OpenFile(
		path.Join(l.logDir, reconcileFailureStreamFile),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		logFilePermissions,
	)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf(
		"Type:%s Account: %s Currency: %s Block: %s:%d computed: %s node: %s\n",
		reconciliationType,
		types.AccountString(account),
		types.CurrencyString(currency),
		block.Hash,
		block.Index,
		computedBalance,
		nodeBalance,
	))
	if err != nil {
		return err
	}

	return nil
}
