package service

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

// initUserTestSchema 在 db 上建最小 DDL：users / roles / user_roles，预置两个角色，并 store.Init。
// 内存库与文件库用例共用。
func initUserTestSchema(t *testing.T, db *sqlx.DB) {
	t.Helper()
	db.MustExec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		account TEXT NOT NULL,
		password TEXT NOT NULL,
		totp_secret TEXT NOT NULL DEFAULT '',
		email TEXT NOT NULL DEFAULT '',
		budget REAL NOT NULL DEFAULT 0,
		unlimited INTEGER NOT NULL DEFAULT 0,
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT (datetime('now', 'localtime'))
	)`)
	db.MustExec(`CREATE UNIQUE INDEX uq_users_account ON users(account)`)
	db.MustExec(`CREATE TABLE roles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL DEFAULT '',
		name TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT (datetime('now', 'localtime'))
	)`)
	db.MustExec(`CREATE TABLE user_roles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		role_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT (datetime('now', 'localtime'))
	)`)
	db.MustExec(`CREATE UNIQUE INDEX uq_user_roles_pair ON user_roles(user_id, role_id)`)
	db.MustExec(`INSERT INTO roles (id, code, name) VALUES (1, 'admin', '管理员'), (2, 'user', '普通用户')`)
	if err := store.Init(db); err != nil {
		t.Fatalf("store init: %v", err)
	}
}

// setupUserTestDB 初始化内存 SQLite（最小 DDL + 预置角色）。
func setupUserTestDB(t *testing.T) {
	t.Helper()
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// 内存库多连接会各自建库，限制单连接
	db.SetMaxOpenConns(1)
	initUserTestSchema(t, db)
}

// countRows 统计表行数，用于校验事务回滚。
func countRows(t *testing.T, table string) int64 {
	t.Helper()
	var n int64
	if err := store.C().Query(`SELECT COUNT(*) FROM `+table, nil).Get(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func newTestUser() *model.User {
	return &model.User{
		Name:      "测试用户",
		Account:   "tester",
		Password:  "hashed-not-plaintext",
		Unlimited: true,
		Enabled:   true,
	}
}

// a) 带角色创建成功，user_roles 有对应行
func TestCreateUserWithRoles_Success(t *testing.T) {
	setupUserTestDB(t)
	u := newTestUser()

	if err := NewUserService().CreateUserWithRoles(u, []int64{1, 2}); err != nil {
		t.Fatalf("CreateUserWithRoles: %v", err)
	}
	if u.ID <= 0 {
		t.Fatalf("user id should be backfilled, got %d", u.ID)
	}
	if n := countRows(t, "user_roles"); n != 2 {
		t.Fatalf("expect 2 user_roles rows, got %d", n)
	}

	roles, err := store.C().Role().ListRolesByUser(u.ID)
	if err != nil {
		t.Fatalf("ListRolesByUser: %v", err)
	}
	if len(roles) != 2 || roles[0].ID != 1 || roles[1].ID != 2 {
		t.Fatalf("unexpected roles: %+v", roles)
	}
}

// b) role_ids 含不存在的 id → ErrRoleNotFound 且事务回滚，users 表无新行
func TestCreateUserWithRoles_RoleNotFoundRollsBack(t *testing.T) {
	setupUserTestDB(t)
	u := newTestUser()

	err := NewUserService().CreateUserWithRoles(u, []int64{1, 999})
	if !errors.Is(err, ErrRoleNotFound) {
		t.Fatalf("expect ErrRoleNotFound, got %v", err)
	}
	if n := countRows(t, "users"); n != 0 {
		t.Fatalf("expect users rollback (0 rows), got %d", n)
	}
	if n := countRows(t, "user_roles"); n != 0 {
		t.Fatalf("expect user_roles rollback (0 rows), got %d", n)
	}
	if u.ID != 0 {
		t.Fatalf("user id should stay zero on rollback, got %d", u.ID)
	}
}

// c) role_ids 为空 → 建出无角色用户，向后兼容
func TestCreateUserWithRoles_EmptyRoles(t *testing.T) {
	setupUserTestDB(t)
	u := newTestUser()

	if err := NewUserService().CreateUserWithRoles(u, nil); err != nil {
		t.Fatalf("CreateUserWithRoles: %v", err)
	}
	if u.ID <= 0 {
		t.Fatalf("user id should be backfilled, got %d", u.ID)
	}
	if n := countRows(t, "users"); n != 1 {
		t.Fatalf("expect 1 user, got %d", n)
	}
	if n := countRows(t, "user_roles"); n != 0 {
		t.Fatalf("expect 0 user_roles rows, got %d", n)
	}
}

// d) 重复 role_ids 不报错，关联只写一行
func TestCreateUserWithRoles_DuplicateRoleIDs(t *testing.T) {
	setupUserTestDB(t)
	u := newTestUser()

	if err := NewUserService().CreateUserWithRoles(u, []int64{1, 1, 1}); err != nil {
		t.Fatalf("CreateUserWithRoles: %v", err)
	}
	if n := countRows(t, "user_roles"); n != 1 {
		t.Fatalf("expect 1 user_roles row, got %d", n)
	}
}

// 账号唯一冲突在事务内仍可被 store.IsUniqueConstraintErr 识别（handler 映射「账号已存在」的依据）
func TestCreateUserWithRoles_DuplicateAccount(t *testing.T) {
	setupUserTestDB(t)
	svc := NewUserService()

	if err := svc.CreateUserWithRoles(newTestUser(), nil); err != nil {
		t.Fatalf("first create: %v", err)
	}
	dup := newTestUser()
	dup.Name = "另一个用户"
	err := svc.CreateUserWithRoles(dup, []int64{1})
	if err == nil || !store.IsUniqueConstraintErr(err) {
		t.Fatalf("expect unique constraint err, got %v", err)
	}
	if n := countRows(t, "users"); n != 1 {
		t.Fatalf("expect 1 user after conflict, got %d", n)
	}
	if n := countRows(t, "user_roles"); n != 0 {
		t.Fatalf("expect 0 user_roles rows after conflict, got %d", n)
	}
}

// 并发回归：守 SQLITE_BUSY_SNAPSHOT。
// 生产 DSN 是 WAL + BEGIN DEFERRED，若事务内"先 SELECT 角色再 INSERT 用户"，
// SELECT 拿只读快照后 INSERT 升级写锁会命中 SQLITE_BUSY_SNAPSHOT（busy_timeout 无效），
// 8 并发下约 40% 请求报 database is locked。修复后（校验移出事务、事务体只写）必须 0 失败。
func TestCreateUserWithRoles_ConcurrentWAL(t *testing.T) {
	const workers = 8
	const rounds = 5

	// 文件库 + 生产同款 DSN，确保真的走 WAL 与 deferred 事务
	dsn := filepath.Join(t.TempDir(), "conc.db") + "?_journal_mode=WAL&_busy_timeout=5000"
	db, err := sqlx.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(8)
	initUserTestSchema(t, db)

	var wg sync.WaitGroup
	var mu sync.Mutex
	failures := make([]string, 0, workers*rounds)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for r := 0; r < rounds; r++ {
				account := fmt.Sprintf("conc-%d-%d", w, r)
				u := newTestUser()
				u.Name = account
				u.Account = account
				if err := NewUserService().CreateUserWithRoles(u, []int64{2}); err != nil {
					mu.Lock()
					failures = append(failures, fmt.Sprintf("worker=%d round=%d: %v", w, r, err))
					mu.Unlock()
				}
			}
		}(w)
	}
	wg.Wait()

	if len(failures) > 0 {
		head := failures
		if len(head) > 5 {
			head = head[:5]
		}
		t.Fatalf("并发创建失败 %d/%d: %v", len(failures), workers*rounds, head)
	}
	if n := countRows(t, "users"); n != workers*rounds {
		t.Fatalf("expect %d users, got %d", workers*rounds, n)
	}
	if n := countRows(t, "user_roles"); n != workers*rounds {
		t.Fatalf("expect %d user_roles rows, got %d", workers*rounds, n)
	}
}
