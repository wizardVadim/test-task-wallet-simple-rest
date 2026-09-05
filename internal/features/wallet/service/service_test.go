package service_test

import (
	"context"
	"errors"
	"testing"

	"wallet-app/internal/core/domain"
	"wallet-app/internal/features/wallet/service"

	"github.com/google/uuid"
)

type repositoryStub struct {
	t            *testing.T
	create       func(context.Context, domain.WalletID) (domain.Wallet, error)
	get          func(context.Context, domain.WalletID) (int64, error)
	getForUpdate func(context.Context, domain.WalletID) (int64, error)
	update       func(context.Context, domain.Wallet) error
	apply        func(context.Context, domain.WalletOperation) error
}

func (r *repositoryStub) CreateNewWallet(ctx context.Context, id domain.WalletID) (domain.Wallet, error) {
	r.t.Helper()
	if r.create == nil {
		r.t.Fatal("unexpected CreateNewWallet call")
	}
	return r.create(ctx, id)
}
func (r *repositoryStub) GetWalletBalance(ctx context.Context, id domain.WalletID) (int64, error) {
	r.t.Helper()
	if r.get == nil {
		r.t.Fatal("unexpected GetWalletBalance call")
	}
	return r.get(ctx, id)
}
func (r *repositoryStub) GetWalletBalanceForUpdate(ctx context.Context, id domain.WalletID) (int64, error) {
	r.t.Helper()
	if r.getForUpdate == nil {
		r.t.Fatal("unexpected GetWalletBalanceForUpdate call")
	}
	return r.getForUpdate(ctx, id)
}
func (r *repositoryStub) UpdateBalance(ctx context.Context, wallet domain.Wallet) error {
	r.t.Helper()
	if r.update == nil {
		r.t.Fatal("unexpected UpdateBalance call")
	}
	return r.update(ctx, wallet)
}
func (r *repositoryStub) ApplyOperation(ctx context.Context, operation domain.WalletOperation) error {
	r.t.Helper()
	if r.apply == nil {
		r.t.Fatal("unexpected ApplyOperation call")
	}
	return r.apply(ctx, operation)
}

type txManagerStub func(context.Context, func(service.Repository) error) error

func (fn txManagerStub) WithinTransaction(ctx context.Context, callback func(service.Repository) error) error {
	return fn(ctx, callback)
}

func mustWalletID(t *testing.T) domain.WalletID {
	t.Helper()
	id, err := domain.NewWalletID(uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestCreateNewWallet(t *testing.T) {
	generatorErr := errors.New("generator failed")
	repositoryErr := errors.New("repository failed")
	for _, tt := range []struct {
		name                                 string
		generatorErr, repositoryErr, wantErr error
		wantRepoCalls                        int
	}{
		{name: "success", wantRepoCalls: 1},
		{name: "generator error", generatorErr: generatorErr, wantErr: generatorErr},
		{name: "repository error", repositoryErr: repositoryErr, wantErr: repositoryErr, wantRepoCalls: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			id := mustWalletID(t)
			want, err := domain.NewWallet(id, 0)
			if err != nil {
				t.Fatal(err)
			}
			generatorCalls, repoCalls := 0, 0
			repo := &repositoryStub{t: t, create: func(gotCtx context.Context, gotID domain.WalletID) (domain.Wallet, error) {
				repoCalls++
				if gotCtx != ctx || gotID != id {
					t.Fatal("incorrect context or wallet ID")
				}
				if tt.repositoryErr != nil {
					return domain.Wallet{}, tt.repositoryErr
				}
				return want, nil
			}}
			svc := service.New(repo, nil, func() (domain.WalletID, error) {
				generatorCalls++
				if tt.generatorErr != nil {
					return domain.WalletID{}, tt.generatorErr
				}
				return id, nil
			})
			got, err := svc.CreateNewWallet(ctx)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v; want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				want = domain.Wallet{}
			}
			if got != want {
				t.Fatalf("wallet = %+v; want %+v", got, want)
			}
			if generatorCalls != 1 || repoCalls != tt.wantRepoCalls {
				t.Fatalf("generator calls = %d, repository calls = %d; want 1, %d", generatorCalls, repoCalls, tt.wantRepoCalls)
			}
		})
	}
}

func TestGetWalletBalance(t *testing.T) {
	repositoryErr := errors.New("read failed")
	for _, tt := range []struct {
		name                   string
		balance                int64
		repositoryErr, wantErr error
		cancelled              bool
	}{
		{name: "positive balance", balance: 100},
		{name: "zero balance"},
		{name: "not found", repositoryErr: domain.ErrWalletNotFound, wantErr: domain.ErrWalletNotFound},
		{name: "repository error", repositoryErr: repositoryErr, wantErr: repositoryErr},
		{name: "cancelled context", cancelled: true, wantErr: context.Canceled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tt.cancelled {
				cancel()
			}
			id := mustWalletID(t)
			calls := 0
			repo := &repositoryStub{t: t, get: func(gotCtx context.Context, gotID domain.WalletID) (int64, error) {
				calls++
				if gotCtx != ctx || gotID != id {
					t.Fatal("incorrect context or wallet ID")
				}
				return tt.balance, tt.repositoryErr
			}}
			got, err := service.New(repo, nil, nil).GetWalletBalance(ctx, id)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v; want %v", err, tt.wantErr)
			}
			if got != tt.balance {
				t.Fatalf("balance = %d; want %d", got, tt.balance)
			}
			wantCalls := 1
			if tt.cancelled {
				wantCalls = 0
			}
			if calls != wantCalls {
				t.Fatalf("repository calls = %d; want %d", calls, wantCalls)
			}
		})
	}
}

// deprecated
// func TestChangeWalletBalance(t *testing.T) {
// 	readErr := errors.New("read failed")
// 	updateErr := errors.New("update failed")
// 	beginErr := errors.New("begin failed")
// 	commitErr := errors.New("commit failed")
// 	for _, tt := range []struct {
// 		name                                             string
// 		operationType                                    domain.OperationType
// 		balance, amount, wantBalance                     int64
// 		readErr, updateErr, beginErr, commitErr, wantErr error
// 		cancelled                                        bool
// 		wantTxCalls, wantReadCalls, wantUpdateCalls      int
// 	}{
// 		{name: "deposit", operationType: domain.OperationTypeDeposit, balance: 100, amount: 50, wantBalance: 150, wantTxCalls: 1, wantReadCalls: 1, wantUpdateCalls: 1},
// 		{name: "deposit to maximum", operationType: domain.OperationTypeDeposit, balance: math.MaxInt64 - 1, amount: 1, wantBalance: math.MaxInt64, wantTxCalls: 1, wantReadCalls: 1, wantUpdateCalls: 1},
// 		{name: "overflow", operationType: domain.OperationTypeDeposit, balance: math.MaxInt64, amount: 1, wantErr: service.ErrBalanceOverflow, wantTxCalls: 1, wantReadCalls: 1},
// 		{name: "withdraw", operationType: domain.OperationTypeWithdraw, balance: 100, amount: 40, wantBalance: 60, wantTxCalls: 1, wantReadCalls: 1, wantUpdateCalls: 1},
// 		{name: "withdraw to zero", operationType: domain.OperationTypeWithdraw, balance: 100, amount: 100, wantTxCalls: 1, wantReadCalls: 1, wantUpdateCalls: 1},
// 		{name: "insufficient funds", operationType: domain.OperationTypeWithdraw, balance: 100, amount: 101, wantErr: service.ErrSmallBalance, wantTxCalls: 1, wantReadCalls: 1},
// 		{name: "not found", operationType: domain.OperationTypeDeposit, amount: 1, readErr: domain.ErrWalletNotFound, wantErr: domain.ErrWalletNotFound, wantTxCalls: 1, wantReadCalls: 1},
// 		{name: "read error", operationType: domain.OperationTypeDeposit, amount: 1, readErr: readErr, wantErr: readErr, wantTxCalls: 1, wantReadCalls: 1},
// 		{name: "update error", operationType: domain.OperationTypeDeposit, amount: 1, wantBalance: 1, updateErr: updateErr, wantErr: updateErr, wantTxCalls: 1, wantReadCalls: 1, wantUpdateCalls: 1},
// 		{name: "begin error", operationType: domain.OperationTypeDeposit, amount: 1, beginErr: beginErr, wantErr: beginErr, wantTxCalls: 1},
// 		{name: "commit error", operationType: domain.OperationTypeDeposit, amount: 1, wantBalance: 1, commitErr: commitErr, wantErr: commitErr, wantTxCalls: 1, wantReadCalls: 1, wantUpdateCalls: 1},
// 		{name: "cancelled context", operationType: domain.OperationTypeDeposit, amount: 1, cancelled: true, wantErr: context.Canceled},
// 	} {
// 		t.Run(tt.name, func(t *testing.T) {
// 			ctx, cancel := context.WithCancel(t.Context())
// 			defer cancel()
// 			if tt.cancelled {
// 				cancel()
// 			}
// 			id := mustWalletID(t)
// 			operation, err := domain.NewWalletOperation(id, tt.operationType, tt.amount)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			txCalls, readCalls, updateCalls := 0, 0, 0
// 			txRepo := &repositoryStub{t: t,
// 				getForUpdate: func(gotCtx context.Context, gotID domain.WalletID) (int64, error) {
// 					readCalls++
// 					if gotCtx != ctx || gotID != id {
// 						t.Fatal("incorrect read context or wallet ID")
// 					}
// 					return tt.balance, tt.readErr
// 				},
// 				update: func(gotCtx context.Context, wallet domain.Wallet) error {
// 					updateCalls++
// 					if readCalls != 1 {
// 						t.Fatal("update must follow the locked read")
// 					}
// 					if gotCtx != ctx || wallet.ID() != id {
// 						t.Fatal("incorrect update context or wallet ID")
// 					}
// 					if wallet.Balance() != tt.wantBalance {
// 						t.Fatalf("updated balance = %d; want %d", wallet.Balance(), tt.wantBalance)
// 					}
// 					return tt.updateErr
// 				},
// 			}
// 			manager := txManagerStub(func(gotCtx context.Context, callback func(service.Repository) error) error {
// 				txCalls++
// 				if gotCtx != ctx {
// 					t.Fatal("incorrect transaction context")
// 				}
// 				if tt.beginErr != nil {
// 					return tt.beginErr
// 				}
// 				if err := callback(txRepo); err != nil {
// 					return err
// 				}
// 				return tt.commitErr
// 			})

// 			svc := service.New(&repositoryStub{t: t}, manager, nil)
// 			if err := svc.ChangeWalletBalance(ctx, operation); !errors.Is(err, tt.wantErr) {
// 				t.Fatalf("error = %v; want %v", err, tt.wantErr)
// 			}
// 			if txCalls != tt.wantTxCalls || readCalls != tt.wantReadCalls || updateCalls != tt.wantUpdateCalls {
// 				t.Fatalf("calls (transaction/read/update) = %d/%d/%d; want %d/%d/%d", txCalls, readCalls, updateCalls, tt.wantTxCalls, tt.wantReadCalls, tt.wantUpdateCalls)
// 			}
// 		})
// 	}
// }

func TestChangeWalletBalance(t *testing.T) {
	applyErr := errors.New("apply operation failed")

	for _, tt := range []struct {
		name      string
		applyErr  error
		wantErr   error
		cancelled bool
		wantCalls int
	}{
		{
			name:      "success",
			wantCalls: 1,
		},
		{
			name:      "repository error",
			applyErr:  applyErr,
			wantErr:   applyErr,
			wantCalls: 1,
		},
		{
			name:      "wallet not found",
			applyErr:  domain.ErrWalletNotFound,
			wantErr:   domain.ErrWalletNotFound,
			wantCalls: 1,
		},
		{
			name:      "insufficient funds",
			applyErr:  service.ErrSmallBalance,
			wantErr:   service.ErrSmallBalance,
			wantCalls: 1,
		},
		{
			name:      "balance overflow",
			applyErr:  service.ErrBalanceOverflow,
			wantErr:   service.ErrBalanceOverflow,
			wantCalls: 1,
		},
		{
			name:      "cancelled context",
			cancelled: true,
			wantErr:   context.Canceled,
			wantCalls: 0,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			if tt.cancelled {
				cancel()
			}

			id := mustWalletID(t)

			operation, err := domain.NewWalletOperation(
				id,
				domain.OperationTypeDeposit,
				100,
			)
			if err != nil {
				t.Fatal(err)
			}

			calls := 0

			repo := &repositoryStub{
				t: t,
				apply: func(
					gotCtx context.Context,
					gotOperation domain.WalletOperation,
				) error {
					calls++

					if gotCtx != ctx {
						t.Fatal("incorrect context")
					}

					if gotOperation != operation {
						t.Fatalf(
							"operation = %+v; want %+v",
							gotOperation,
							operation,
						)
					}

					return tt.applyErr
				},
			}

			svc := service.New(repo, nil, nil)

			err = svc.ChangeWalletBalance(ctx, operation)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v; want %v", err, tt.wantErr)
			}

			if calls != tt.wantCalls {
				t.Fatalf(
					"ApplyOperation calls = %d; want %d",
					calls,
					tt.wantCalls,
				)
			}
		})
	}
}
