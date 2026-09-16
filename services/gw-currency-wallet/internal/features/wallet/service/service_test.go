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
	t      *testing.T
	create func(context.Context, domain.WalletID) (domain.Wallet, error)
	get    func(context.Context, domain.WalletID) (int64, error)
	apply  func(context.Context, domain.WalletOperation) error
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
func (r *repositoryStub) ApplyOperation(ctx context.Context, operation domain.WalletOperation) error {
	r.t.Helper()
	if r.apply == nil {
		r.t.Fatal("unexpected ApplyOperation call")
	}
	return r.apply(ctx, operation)
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
			svc := service.New(repo, func() (domain.WalletID, error) {
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
			got, err := service.New(repo, nil).GetWalletBalance(ctx, id)
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
			applyErr:  domain.ErrSmallBalance,
			wantErr:   domain.ErrSmallBalance,
			wantCalls: 1,
		},
		{
			name:      "balance overflow",
			applyErr:  domain.ErrBalanceOverflow,
			wantErr:   domain.ErrBalanceOverflow,
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

			svc := service.New(repo, nil)

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
