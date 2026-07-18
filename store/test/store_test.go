package test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/driver"
	"github.com/lijcoder/aiapi/store/model"
)

func setup(t *testing.T) (cleanup func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// driver 要求数据库文件已存在，先创建空文件
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	f, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("create db failed: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close db failed: %v", err)
	}

	db, err := driver.NewSQLite(dbPath)
	if err != nil {
		t.Fatalf("create db driver failed: %v", err)
	}
	if err := store.Init(db); err != nil {
		t.Fatalf("store init failed: %v", err)
	}

	// 建表
	schema, err := os.ReadFile(filepath.Join("..", "..", "sql", "sqlite.sql"))
	if err != nil {
		t.Fatalf("read schema failed: %v", err)
	}
	for _, stmt := range strings.Split(string(schema), ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		if _, err := store.C().RawExec(stmt); err != nil {
			t.Fatalf("exec schema failed: %v", err)
		}
	}

	return func() {
		if err := store.Close(); err != nil {
			t.Fatalf("store close failed: %v", err)
		}
	}
}

func TestSessionGetUserByID(t *testing.T) {
	cleanup := setup(t)
	defer cleanup()

	// 插入测试用户
	if _, err := store.C().RawExec(
		`INSERT INTO users (name, account, budget, unlimited, enabled) VALUES (?, ?, ?, ?, ?)`,
		"test", "test", 100, 0, 1,
	); err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	user, err := store.C().GetUserByID(1)
	if err != nil {
		t.Fatalf("get user failed: %v", err)
	}
	if user == nil {
		t.Fatalf("user not found")
	}
	if user.Name != "test" {
		t.Errorf("want name=test, got %s", user.Name)
	}
	if user.Budget != 100 {
		t.Errorf("want budget=100, got %f", user.Budget)
	}
}

func TestSessionTransaction(t *testing.T) {
	cleanup := setup(t)
	defer cleanup()

	// 插入测试用户
	if _, err := store.C().RawExec(
		`INSERT INTO users (name, account, budget, unlimited, enabled) VALUES (?, ?, ?, ?, ?)`,
		"tx_user", "tx_user", 50, 0, 1,
	); err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	// 事务内充值
	err := store.T(func(s *store.Session) error {
		user, err := s.GetUserByID(1)
		if err != nil {
			return err
		}
		if user == nil {
			return errors.New("user not found")
		}

		rec := &model.RechargeRecord{
			UserID:        user.ID,
			Amount:        50,
			BalanceBefore: user.Budget,
			BalanceAfter:  user.Budget + 50,
			Operator:      "admin",
			Remark:        "test recharge",
		}
		if err := s.InsertRechargeRecord(rec); err != nil {
			return err
		}
		return s.UpdateUserBudget(user.ID, rec.BalanceAfter)
	})
	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}

	user, err := store.C().GetUserByID(1)
	if err != nil {
		t.Fatalf("get user after tx failed: %v", err)
	}
	if user.Budget != 100 {
		t.Errorf("want budget=100 after tx, got %f", user.Budget)
	}
}

func TestSessionTransactionRollback(t *testing.T) {
	cleanup := setup(t)
	defer cleanup()

	// 插入测试用户
	if _, err := store.C().RawExec(
		`INSERT INTO users (name, account, budget, unlimited, enabled) VALUES (?, ?, ?, ?, ?)`,
		"rollback_user", "rollback_user", 50, 0, 1,
	); err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	// 事务内失败，应回滚
	err := store.T(func(s *store.Session) error {
		if err := s.UpdateUserBudget(1, 999); err != nil {
			return err
		}
		return errors.New("rollback")
	})
	if err == nil {
		t.Fatalf("expected rollback error")
	}

	user, err := store.C().GetUserByID(1)
	if err != nil {
		t.Fatalf("get user after rollback failed: %v", err)
	}
	if user.Budget != 50 {
		t.Errorf("want budget=50 after rollback, got %f", user.Budget)
	}
}
