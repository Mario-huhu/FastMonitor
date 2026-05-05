package process

import "runtime"

// IProcessMapper 进程映射器接口
type IProcessMapper interface {
	// GetPIDByConnection 通过五元组获取PID
	GetPIDByConnection(protocol, srcIP, dstIP string, srcPort, dstPort uint32) (int32, *ProcessInfo, bool)

	// GetPIDByPort 通过端口获取PID
	GetPIDByPort(protocol string, port uint32) (int32, *ProcessInfo, bool)

	// GetStats 获取统计信息
	GetStats() map[string]interface{}

	// Update 更新映射表
	Update() error

	// Stop 停止映射器
	Stop()
}

// NewPlatformProcessMapper 创建平台特定的进程映射器
func NewPlatformProcessMapper() IProcessMapper {
	switch runtime.GOOS {
	case "darwin":
		// macOS: 使用 libproc 原生 API
		return createDarwinMapper()
	default:
		// Windows/Linux: 使用标准实现
		return NewProcessMapper()
	}
}

// createDarwinMapper 创建 macOS 映射器的占位函数
// 在非 darwin 平台上回退到标准实现
// 在 darwin 平台上会被 mapper_darwin_impl.go 中的实现覆盖
var createDarwinMapper = func() IProcessMapper {
	return NewProcessMapper()
}
