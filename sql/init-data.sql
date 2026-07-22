-- 初始数据 — 手动执行: sqlite3 ~/.aiapi/aiapi.db < sql/init-data.sql
-- 依赖 sqlite.sql 中的表结构，需先执行 DDL

-- 建角色
INSERT OR IGNORE INTO roles (id, code, name) VALUES (1, 'admin', '管理员');
INSERT OR IGNORE INTO roles (id, code, name) VALUES (2, 'user',  '普通用户');

-- 建权限（entity='API', action='*'）
-- admin 角色（id=1）超管特权：value='*' 放行所有接口
INSERT OR IGNORE INTO role_permission (role_id, entity, action, value) VALUES
  (1, 'API', '*', '*');
-- user 角色权限（id=2）按接口路径精确授权
INSERT OR IGNORE INTO role_permission (role_id, entity, action, value) VALUES
  (2, 'API', '*', '/manager/self'),
  (2, 'API', '*', '/manager/recharge/self'),
  (2, 'API', '*', '/manager/recharge/records/self'),
  (2, 'API', '*', '/manager/apikeys/list/self'),
  (2, 'API', '*', '/manager/apikeys/create/self'),
  (2, 'API', '*', '/manager/apikeys/toggle/self'),
  (2, 'API', '*', '/manager/apikeys/delete/self'),
  (2, 'API', '*', '/manager/apikeys/rename/self'),
  (2, 'API', '*', '/manager/apikeys/budget/self'),
  (2, 'API', '*', '/manager/usage/stats/self'),
  (2, 'API', '*', '/manager/usage/filters/self');

-- 建菜单
INSERT OR IGNORE INTO menus (id, parent_id, name, path, sort_order) VALUES
  (1,  0, '使用统计',     '/usage',           1),
  (2,  0, 'API 密钥',     '/apikeys',         2),
  (3,  0, '模型列表',     '/models',          3),
  (4,  0, '充值中心',     '/recharge',        4),
  (5,  0, '仪表盘',       '/admin/dashboard', 5),
  (6,  0, '用户管理',     '/admin/users',     6),
  (7,  0, '提供商管理', '/admin/providers', 7),
  (8,  0, '模型管理',     '/admin/models',    8),
  (9,  0, '全局统计',     '/admin/usage',     9),
  (10, 0, '充值记录',     '/admin/recharge',  10);

-- 存量库菜单名升级（原「模型定价」改名为「模型管理」）：
--   UPDATE menus SET name='模型管理' WHERE id=8;

-- 分配菜单给角色
-- admin 角色（id=1）→ 全部菜单
INSERT OR IGNORE INTO role_menus (role_id, menu_id) VALUES
  (1, 1), (1, 2), (1, 3), (1, 4), (1, 5), (1, 6), (1, 7), (1, 8), (1, 9), (1, 10);
-- user 角色（id=2）→ 自助菜单
INSERT OR IGNORE INTO role_menus (role_id, menu_id) VALUES
  (2, 1), (2, 2), (2, 3), (2, 4);

-- 建管理员账号并分配角色（password 用 bcrypt 生成，替换 <bcrypt-hash>）
-- INSERT INTO users (name, account, password, unlimited, enabled) VALUES ('管理员', 'admin', '<bcrypt-hash>', 1, 1);
-- INSERT INTO user_roles (user_id, role_id) VALUES ((SELECT id FROM users WHERE account='admin'), 1);
