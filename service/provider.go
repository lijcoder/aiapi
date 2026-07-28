// provider.go Provider 配置相关业务。
//
// providers.config 含上游真实 API Key（headers 里），落库前整体 AES-GCM 加密
// （通用加密能力见 crypto.go），DB 泄露不直接暴露上游凭证。
//
// 兼容存量明文：历史行为明文 JSON（以 '{' 开头），读取时识别并原样放行，
// 管理端重新保存后自动转为密文（渐进迁移，无需 DDL/DML）。
//
// 读侧统一入口：各业务不要自行「查库 → 解密 → ParseConfig」三段拼装，
// 单个用 GetProviderConfig（含查库），已有 provider 记录用 ParseProviderConfig。
package service

import (
	"strings"

	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

// EncryptProviderConfig 加密 provider config JSON，返回可落库的密文。
func EncryptProviderConfig(cfgJSON string) (string, error) {
	return encryptWithPurpose(cfgJSON, purposeProviderConfig)
}

// EncryptProvider 加密 provider 对象的敏感字段（Config，含上游 Key），原地替换为密文。
// 上层组好对象后先调本方法再落库（Create/Update），无需感知加密细节。
// 已是密文的 Config 跳过（防重复加密导致无法解密）。
func EncryptProvider(p *model.Provider) error {
	if !strings.HasPrefix(strings.TrimSpace(p.Config), "{") {
		return nil // 已是密文
	}
	enc, err := EncryptProviderConfig(p.Config)
	if err != nil {
		return err
	}
	p.Config = enc
	return nil
}

// DecryptProviderConfig 解密读取到的 provider config。
// 存量明文（'{' 开头的 JSON）原样返回，实现渐进迁移。
func DecryptProviderConfig(stored string) (string, error) {
	if strings.HasPrefix(strings.TrimSpace(stored), "{") {
		return stored, nil // 存量明文，放行
	}
	return decryptWithPurpose(stored, purposeProviderConfig)
}

// ParseProviderConfig 解密并解析 provider 的 config（纯解析，不查库）。
// 适用于调用方已持有 provider 记录的场景（如管理端列表逐行解析）。
func ParseProviderConfig(pvd *model.Provider) (*model.ProviderConfig, error) {
	plain, err := DecryptProviderConfig(pvd.Config)
	if err != nil {
		return nil, err
	}
	pvd.Config = plain
	return pvd.ParseConfig()
}

// GetProviderConfig 按类型查询「启用中」的 provider 并返回解密解析后的配置，一次完成。
// proxy 转发等场景的统一入口，调用方无需再处理解密/解析细节。
//
// 返回值约定：
//   - (nil, nil, err)   数据库或内部错误
//   - (nil, nil, nil)   provider 不存在或已禁用
//   - (pvd, nil, nil)   provider 存在但 config 解密/解析失败（配置损坏）
//   - (pvd, cfg, nil)   正常
func GetProviderConfig(pvdType string) (*model.Provider, *model.ProviderConfig, error) {
	pvd, err := store.C().Provider().Get(pvdType)
	if err != nil || pvd == nil {
		return nil, nil, err
	}
	cfg, err := ParseProviderConfig(pvd)
	if err != nil {
		return pvd, nil, nil
	}
	return pvd, cfg, nil
}
