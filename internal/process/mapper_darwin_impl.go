// +build darwin,cgo

package process

// 在 darwin 平台上使用原生实现
func init() {
	createDarwinMapper = func() IProcessMapper {
		return NewDarwinProcessMapper()
	}
}

// 确保 DarwinProcessMapper 实现 IProcessMapper 接口
var _ IProcessMapper = (*DarwinProcessMapper)(nil)
