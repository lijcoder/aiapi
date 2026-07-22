package store

import (
	"database/sql"
	"errors"

	"github.com/lijcoder/aiapi/store/model"
)

// User 返回用户相关操作的命名空间。
func (s *Session) User() *UserStore {
	return &UserStore{s: s}
}

// UserStore 是用户相关操作的命名空间。
type UserStore struct {
	s *Session
}

// GetByID 按 ID 查询启用中的用户
func (us *UserStore) GetByID(id int64) (*model.User, error) {
	var u model.User
	err := us.s.Query(
		`SELECT * FROM users WHERE id = :id AND enabled = 1`,
		map[string]any{"id": id},
	).Get(&u)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByAccount 按账号查询用户（不过滤 enabled，调用方自行判断启用状态）
func (us *UserStore) GetByAccount(account string) (*model.User, error) {
	var u model.User
	err := us.s.Query(
		`SELECT * FROM users WHERE account = :account`,
		map[string]any{"account": account},
	).Get(&u)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdatePassword 更新用户密码哈希
func (us *UserStore) UpdatePassword(userID int64, passwordHash string) error {
	_, err := us.s.Query(
		`UPDATE users SET password = :password WHERE id = :user_id`,
		map[string]any{"user_id": userID, "password": passwordHash},
	).Exec()
	return err
}

// SetEnabled 启用/禁用用户
func (us *UserStore) SetEnabled(userID int64, enabled bool) error {
	_, err := us.s.Query(
		`UPDATE users SET enabled = :enabled WHERE id = :user_id`,
		map[string]any{"user_id": userID, "enabled": enabled},
	).Exec()
	return err
}

// List 列出全部用户（按 id 倒序）。
func (us *UserStore) List() ([]model.User, error) {
	var users []model.User
	err := us.s.Query(
		`SELECT * FROM users ORDER BY id DESC`,
		nil,
	).Select(&users)
	if err != nil {
		return nil, err
	}
	return users, nil
}

// Count 用户总数
func (us *UserStore) Count() (int64, error) {
	var n int64
	err := us.s.Query(`SELECT COUNT(*) FROM users`, nil).Get(&n)
	return n, err
}

// Create 创建用户，回填 ID。
func (us *UserStore) Create(u *model.User) error {
	res, err := us.s.Query(
		`INSERT INTO users (name, account, password, budget, unlimited, enabled)
		 VALUES (:name, :account, :password, :budget, :unlimited, :enabled)`,
		map[string]any{
			"name":      u.Name,
			"account":   u.Account,
			"password":  u.Password,
			"budget":    u.Budget,
			"unlimited": u.Unlimited,
			"enabled":   u.Enabled,
		},
	).Exec()
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	u.ID = id
	return nil
}

// UpdateProfile 更新用户基本信息（姓名、额度、限额模式），账号不可改。
func (us *UserStore) UpdateProfile(u *model.User) error {
	_, err := us.s.Query(
		`UPDATE users SET name=:name, budget=:budget, unlimited=:unlimited
		 WHERE id=:id`,
		map[string]any{
			"id":        u.ID,
			"name":      u.Name,
			"budget":    u.Budget,
			"unlimited": u.Unlimited,
		},
	).Exec()
	return err
}

// UpdateProfileSelf 用户自助更新姓名和邮箱（不动额度/限额/账号/密码）
func (us *UserStore) UpdateProfileSelf(userID int64, name, email string) error {
	_, err := us.s.Query(
		`UPDATE users SET name=:name, email=:email WHERE id=:id`,
		map[string]any{"id": userID, "name": name, "email": email},
	).Exec()
	return err
}

// GetByIDAny 按 ID 查询用户（不过滤 enabled），管理员场景使用。
func (us *UserStore) GetByIDAny(id int64) (*model.User, error) {
	var u model.User
	err := us.s.Query(
		`SELECT * FROM users WHERE id = :id`,
		map[string]any{"id": id},
	).Get(&u)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
