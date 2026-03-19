package suite

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"grpc-service-ref/internal/app"
	"grpc-service-ref/internal/config"

	ssov1 "github.com/JustSkiv/protos/gen/go/sso"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Suite struct {
	*testing.T
	Cfg        *config.Config
	AuthClient ssov1.AuthClient
}

const (
	grpcHost           = "127.0.0.1"
	serverStartTimeout = 5 * time.Second
	defaultRPCTimeout  = 15 * time.Second
)

// New creates new test suite.
func New(t *testing.T) (context.Context, *Suite) {
	t.Helper()

	cfg := config.MustLoadPath(configPath())
	cfg.StoragePath = filepath.Join(t.TempDir(), "sso.db")
	cfg.GRPC.Port = freePort(t)

	applyMigrations(t, cfg.StoragePath, migrationsPath())

	application := app.New(newDiscardLogger(), cfg.GRPC.Port, cfg.StoragePath, cfg.TokenTTL)
	go application.GRPCServer.MustRun()
	t.Cleanup(func() {
		_ = application.Stop()
	})

	cc := waitForGRPC(t, grpcAddress(cfg))
	t.Cleanup(func() {
		_ = cc.Close()
	})

	timeout := cfg.GRPC.Timeout
	if timeout <= 0 || timeout > time.Minute {
		timeout = defaultRPCTimeout
	}

	ctx, cancelCtx := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(func() {
		t.Helper()
		cancelCtx()
	})

	return ctx, &Suite{
		T:          t,
		Cfg:        cfg,
		AuthClient: ssov1.NewAuthClient(cc),
	}
}

func configPath() string {
	const key = "CONFIG_PATH"

	if v := os.Getenv(key); v != "" {
		return v
	}

	return "../config/local_tests.yaml"
}

func migrationsPath() string {
	const key = "MIGRATIONS_PATH"

	if v := os.Getenv(key); v != "" {
		return v
	}

	return "../migrations"
}

func grpcAddress(cfg *config.Config) string {
	return net.JoinHostPort(grpcHost, strconv.Itoa(cfg.GRPC.Port))
}

func freePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", net.JoinHostPort(grpcHost, "0"))
	if err != nil {
		t.Fatalf("failed to allocate free tcp port: %v", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	return listener.Addr().(*net.TCPAddr).Port
}

func applyMigrations(t *testing.T, storagePath, migrationsPath string) {
	t.Helper()

	m, err := migrate.New(
		"file://"+filepath.ToSlash(migrationsPath),
		fmt.Sprintf("sqlite://%s", filepath.ToSlash(storagePath)),
	)
	if err != nil {
		t.Fatalf("failed to initialize migrations: %v", err)
	}

	t.Cleanup(func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			t.Logf("failed to close migration source: %v", srcErr)
		}
		if dbErr != nil {
			t.Logf("failed to close migration db: %v", dbErr)
		}
	})

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("failed to apply migrations: %v", err)
	}
}

func waitForGRPC(t *testing.T, address string) *grpc.ClientConn {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), serverStartTimeout)
	defer cancel()

	cc, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("grpc server connection failed: %v", err)
	}

	return cc
}

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
