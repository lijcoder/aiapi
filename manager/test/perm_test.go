package test

import (
	"testing"

	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/store/model"
)

func grant(roleID int64, entity, action, value string) model.RolePermission {
	return model.RolePermission{RoleID: roleID, Entity: entity, Action: action, Value: value}
}

func TestCheckPathPermission_Granted(t *testing.T) {
	perms := []model.RolePermission{
		grant(1, base.EntityAPI, "*", "/manager/me"),
		grant(1, base.EntityAPI, "*", "/manager/recharge"),
	}
	if err := base.CheckPathPermission(perms, "/manager/me"); err != nil {
		t.Fatalf("expected pass for granted path, got %v", err)
	}
	if err := base.CheckPathPermission(perms, "/manager/recharge"); err != nil {
		t.Fatalf("expected pass for granted path, got %v", err)
	}
}

func TestCheckPathPermission_NotGranted(t *testing.T) {
	perms := []model.RolePermission{
		grant(2, base.EntityAPI, "*", "/manager/apikeys/budget/self"),
	}
	if err := base.CheckPathPermission(perms, "/manager/apikeys/budget"); err != base.ErrForbidden {
		t.Fatalf("expected ErrForbidden for ungranted path, got %v", err)
	}
}

func TestCheckPathPermission_OtherEntityIgnored(t *testing.T) {
	perms := []model.RolePermission{
		{RoleID: 1, Entity: "MENU", Action: "*", Value: "/manager/me"},
	}
	if err := base.CheckPathPermission(perms, "/manager/me"); err != base.ErrForbidden {
		t.Fatalf("expected ErrForbidden when entity != API, got %v", err)
	}
}

func TestCheckPathPermission_EmptyPerms(t *testing.T) {
	if err := base.CheckPathPermission(nil, "/manager/me"); err != base.ErrForbidden {
		t.Fatalf("expected ErrForbidden for empty perms, got %v", err)
	}
}

func TestCheckPathPermission_UnionAcrossRoles(t *testing.T) {
	perms := []model.RolePermission{
		grant(1, base.EntityAPI, "*", "/manager/apikeys/budget"),
		grant(2, base.EntityAPI, "*", "/manager/apikeys/budget/self"),
	}
	if err := base.CheckPathPermission(perms, "/manager/apikeys/budget"); err != nil {
		t.Fatalf("expected pass via role 1, got %v", err)
	}
	if err := base.CheckPathPermission(perms, "/manager/apikeys/budget/self"); err != nil {
		t.Fatalf("expected pass via role 2, got %v", err)
	}
	// 路径精确匹配，同前缀不串权：充值只对超管开放，普通用户角色没有 /manager/recharge
	if err := base.CheckPathPermission(perms, "/manager/recharge"); err != base.ErrForbidden {
		t.Fatalf("expected ErrForbidden for ungranted path, got %v", err)
	}
}
