package devices

import (
	"fmt"

	"github.com/gen2brain/malgo"
)

// DeviceInfo 设备信息
type DeviceInfo struct {
	ID      malgo.DeviceID
	Name    string
	Formats []malgo.DataFormat
	Error   string
}

// ListPlaybackDevices 列出所有播放设备
func ListPlaybackDevices(ctx *malgo.AllocatedContext) ([]DeviceInfo, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}

	infos, err := ctx.Devices(malgo.Playback)
	if err != nil {
		return nil, fmt.Errorf("获取播放设备列表失败: %w", err)
	}

	result := make([]DeviceInfo, 0, len(infos))
	for _, info := range infos {
		deviceInfo := DeviceInfo{
			ID:   info.ID,
			Name: info.Name(),
		}

		full, err := ctx.DeviceInfo(malgo.Playback, info.ID, malgo.Shared)
		if err != nil {
			deviceInfo.Error = err.Error()
		} else {
			deviceInfo.Formats = full.Formats
		}

		result = append(result, deviceInfo)
	}

	return result, nil
}

// ListCaptureDevices 列出所有捕获设备
func ListCaptureDevices(ctx *malgo.AllocatedContext) ([]DeviceInfo, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}

	infos, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return nil, fmt.Errorf("获取捕获设备列表失败: %w", err)
	}

	result := make([]DeviceInfo, 0, len(infos))
	for _, info := range infos {
		deviceInfo := DeviceInfo{
			ID:   info.ID,
			Name: info.Name(),
		}

		full, err := ctx.DeviceInfo(malgo.Capture, info.ID, malgo.Shared)
		if err != nil {
			deviceInfo.Error = err.Error()
		} else {
			deviceInfo.Formats = full.Formats
		}

		result = append(result, deviceInfo)
	}

	return result, nil
}

// PrintPlaybackDevices 打印所有播放设备信息
func PrintPlaybackDevices(ctx *malgo.AllocatedContext) error {
	devices, err := ListPlaybackDevices(ctx)
	if err != nil {
		return err
	}

	fmt.Println("Playback Devices:")
	for i, device := range devices {
		status := "ok"
		if device.Error != "" {
			status = device.Error
		}
		fmt.Printf("    %d: %v, %s, [%s], formats: %+v\n",
			i, device.ID, device.Name, status, device.Formats)
	}

	return nil
}

// PrintCaptureDevices 打印所有捕获设备信息
func PrintCaptureDevices(ctx *malgo.AllocatedContext) error {
	devices, err := ListCaptureDevices(ctx)
	if err != nil {
		return err
	}

	fmt.Println("Capture Devices:")
	for i, device := range devices {
		status := "ok"
		if device.Error != "" {
			status = device.Error
		}
		fmt.Printf("    %d: %v, %s, [%s], formats: %+v\n",
			i, device.ID, device.Name, status, device.Formats)
	}

	return nil
}

// PrintAllDevices 打印所有设备信息（播放和捕获）
func PrintAllDevices(ctx *malgo.AllocatedContext) error {
	if err := PrintPlaybackDevices(ctx); err != nil {
		return err
	}
	fmt.Println()
	if err := PrintCaptureDevices(ctx); err != nil {
		return err
	}
	return nil
}
