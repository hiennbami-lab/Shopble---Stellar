package database

import (
	"context"

	"shopble/common/comerr"
	"shopble/common/comlog"

	"cloud.google.com/go/logging"
	"gorm.io/gorm"
)

type (
	TxnWrapper struct {
		db                *gorm.DB
		log               *comlog.OurLog
		onCommitCallbacks []TxnOnCallback
	}

	TxnCallback   = func(ctx context.Context) TxnDbCallback
	TxnDbCallback = func(db *gorm.DB) error
	TxnOnCallback = func() error
)

func newTxnWrapper() (*TxnWrapper, error) {
	sqlDb := GetDb()
	return &TxnWrapper{
		db:                sqlDb.DB,
		log:               comlog.GetLog(),
		onCommitCallbacks: make([]TxnOnCallback, 0),
	}, nil
}

func (txWrapper *TxnWrapper) executeTx(ctx context.Context, callback TxnCallback) error {
	var (
		ctxKey = tContextKey("sql:tx")
		db     *gorm.DB
	)
	ctxTx := ctx.Value(ctxKey)
	if ctxTx != nil {
		db = ctxTx.(*gorm.DB)
	} else {
		db = GetDb().DB
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, ctxKey, tx)
		return callback(txCtx)(tx)
	}); err != nil {
		return err
	}
	txWrapper.executeCallbacks(txWrapper.onCommitCallbacks, "commit")
	return nil
}

func (txWrapper *TxnWrapper) executeCallbacks(callbacks []TxnOnCallback, name string) {
	if len(callbacks) > 0 {
		for _, callback := range callbacks {
			execOnCommitCallback(callback)
		}
	}
}

func Txn(ctx context.Context, execFunc TxnCallback) (err error) {
	var (
		ctxKey        = tContextKey("sql:tx_wrapper")
		ctxTxnWrapper = ctx.Value(ctxKey)
		txnWrapper    *TxnWrapper
	)
	if ctxTxnWrapper != nil {
		txnWrapper = ctxTxnWrapper.(*TxnWrapper)
	} else {
		txnWrapper, err = newTxnWrapper()
		if err != nil {
			return err
		}
		ctx = context.WithValue(ctx, ctxKey, txnWrapper)
	}
	return txnWrapper.executeTx(ctx, execFunc)
}

func OnTxCommit(ctx context.Context, handler TxnOnCallback) {
	var (
		ctxKey    = tContextKey("sql:tx_wrapper")
		txWrapper = ctx.Value(ctxKey)
	)
	if txWrapper == nil {
		execOnCommitCallback(handler)
		return
	}
	txWrapper.(*TxnWrapper).RegisterOnCommitCallback(handler)
}

func execOnCommitCallback(handler TxnOnCallback) {
	var (
		ourLog = comlog.GetLog()
		err    = handler()
	)
	if err != nil {
		if ourErr, ok := err.(comerr.Failure); ok {
			ourLog.Log(logging.Entry{
				Payload: map[string]any{
					"callback_type": "onCommit",
					"error":         ourErr.Error(),
					"stacktrace":    ourErr.Stack(),
					"data":          ourErr.Data().String(),
				},
				Severity: logging.Error,
			})
		} else {
			ourLog.Log(logging.Entry{
				Payload: map[string]any{
					"unexpected_error": true,
					"callback_type":    "onCommit",
					"error":            err.Error(),
				},
				Severity: logging.Error,
			})
		}
	}
}

func (txWrapper *TxnWrapper) RegisterOnCommitCallback(callback TxnOnCallback) {
	txWrapper.onCommitCallbacks = append(txWrapper.onCommitCallbacks, callback)
}
