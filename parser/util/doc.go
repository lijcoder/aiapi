// Package util 提供各协议解析器共用的纯工具：SSE 行与事件切分、请求头取 API Key、
// 请求体顶层模型名改写。
//
// 本包只放协议无关、无状态的纯函数，且不反向 import parser 包（由 parser 引用本包），
// 避免协议解析层与工具层互相依赖。
package util
