package base

import "errors"

// 启动期配置错误
var ErrJWTSecretInvalid = errors.New("AIAPI_JWT_SECRET environment variable must be set and at least 32 bytes")
