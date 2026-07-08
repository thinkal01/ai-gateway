package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/vrviu/ai-gateway/internal/model"
	"github.com/vrviu/ai-gateway/internal/store"
)

// Service 认证服务
type Service struct {
	keyStore     store.ApiKeyStore
	prefixLength int
}

// NewService 创建认证服务
func NewService(keyStore store.ApiKeyStore, prefixLength int) *Service {
	return &Service{
		keyStore:     keyStore,
		prefixLength: prefixLength,
	}
}

// ValidateKey 校验 API Key
// 输入: "sk-abc123..." 完整 Key 字符串
// 输出: AuthResult 包含租户ID、KeyID、Scopes
func (s *Service) ValidateKey(keyStr, modelName string) (*model.AuthResult, error) {
	// 1. 提取前缀用于查找
	prefix := extractPrefix(keyStr, s.prefixLength)
	if prefix == "" {
		return nil, fmt.Errorf("invalid key format")
	}

	// 2. 查找 Key
	apiKey, err := s.keyStore.FindByPrefix(prefix)
	if err != nil {
		return nil, fmt.Errorf("find key: %w", err)
	}
	if apiKey == nil {
		return nil, fmt.Errorf("key not found")
	}

	// 3. 验证哈希
	hash := hashKey(keyStr)
	if hash != apiKey.KeyHash {
		return nil, fmt.Errorf("invalid key")
	}

	// 4. 验证启用状态
	if !apiKey.IsActive {
		return nil, fmt.Errorf("key is disabled")
	}

	// 5. 验证过期时间（零值永不过期）
	if !apiKey.ExpiresAt.IsZero() && time.Now().UTC().After(apiKey.ExpiresAt) {
		return nil, fmt.Errorf("key has expired")
	}

	// 6. 验证 Scope
	if !MatchScope(apiKey.Scopes, modelName) {
		return nil, fmt.Errorf("insufficient permissions for model %s", modelName)
	}

	// 7. 更新最后使用时间（异步不阻塞）
	go func() {
		_ = s.keyStore.UpdateLastUsed(apiKey.ID, time.Now().UTC())
	}()

	return &model.AuthResult{
		TenantID: apiKey.TenantID,
		KeyID:    apiKey.ID,
		Scopes:   apiKey.Scopes,
	}, nil
}

// extractPrefix 提取 Key 的前 N 位作为前缀，自动跳过 "sk-" 前缀
func extractPrefix(keyStr string, length int) string {
	keyStr = strings.TrimSpace(keyStr)
	// 去掉 "sk-" 前缀
	keyStr = strings.TrimPrefix(keyStr, "sk-")
	if len(keyStr) < length {
		return ""
	}
	return keyStr[:length]
}

// hashKey 对 Key 进行 SHA256 哈希
func hashKey(keyStr string) string {
	h := sha256.Sum256([]byte(keyStr))
	return hex.EncodeToString(h[:])
}

// GenerateKey 生成新的 API Key
// 格式: sk-{prefix}{secret}
func GenerateKey(prefixLength, secretLength int) (fullKey, prefix, hash string, err error) {
	// 生成随机字节
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", "", fmt.Errorf("generate random bytes: %w", err)
	}

	hexStr := hex.EncodeToString(b)
	prefix = hexStr[:prefixLength]

	fullKey = "sk-" + prefix + hexStr[:secretLength*2]
	hash = hashKey(fullKey)

	return fullKey, prefix, hash, nil
}
