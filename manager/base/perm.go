package base

import (
	"errors"

	"github.com/lijcoder/aiapi/store/model"
)

// EntityAPI 权限实体类型：接口
const EntityAPI = "API"

// ErrForbidden 权限不足
var ErrForbidden = errors.New("forbidden")

// CheckPathPermission 接口级权限判定：
// - 用户任一角色拥有 (entity=API, value=path) 即通过；
// - 用户任一角色拥有 (entity=API, value=*) 则放行所有接口（超管特权）；
// action 字段暂按通配处理（保留供后续细粒度）。
func CheckPathPermission(perms []model.RolePermission, path string) error {
	for _, p := range perms {
		if p.Entity != EntityAPI {
			continue
		}
		if p.Value == "*" || p.Value == path {
			return nil
		}
	}
	return ErrForbidden
}
