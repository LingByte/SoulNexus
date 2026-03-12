package devices

//
//func TestDefaultDeviceConfig(t *testing.T) {
//	config := DefaultDeviceConfig()
//	if config.SampleRate != 44100 {
//		t.Errorf("期望采样率为 44100，实际为 %d", config.SampleRate)
//	}
//	if config.Channels != 1 {
//		t.Errorf("期望声道数为 1，实际为 %d", config.Channels)
//	}
//	if config.Format != malgo.FormatS16 {
//		t.Errorf("期望格式为 FormatS16，实际为 %v", config.Format)
//	}
//	if config.AlsaNoMMap != 1 {
//		t.Errorf("期望 AlsaNoMMap 为 1，实际为 %d", config.AlsaNoMMap)
//	}
//}
//
//func TestNewAudioDevice(t *testing.T) {
//	config := DefaultDeviceConfig()
//	config.LogCallback = func(message string) {
//		// 测试时不输出日志
//	}
//
//	device, err := NewAudioDevice(config)
//	if err != nil {
//		t.Fatalf("创建音频设备失败: %v", err)
//	}
//	defer device.Close()
//
//	if device == nil {
//		t.Fatal("设备不应该为 nil")
//	}
//
//	if device.ctx == nil {
//		t.Fatal("音频上下文不应该为 nil")
//	}
//
//	if device.config == nil {
//		t.Fatal("配置不应该为 nil")
//	}
//}
//
//func TestNewAudioDeviceWithNilConfig(t *testing.T) {
//	device, err := NewAudioDevice(nil)
//	if err != nil {
//		t.Fatalf("使用 nil 配置创建音频设备失败: %v", err)
//	}
//	defer device.Close()
//
//	if device.config.SampleRate != 44100 {
//		t.Errorf("期望默认采样率为 44100，实际为 %d", device.config.SampleRate)
//	}
//}
//
//func TestAudioDeviceRecording(t *testing.T) {
//	config := DefaultDeviceConfig()
//	config.LogCallback = func(message string) {}
//
//	device, err := NewAudioDevice(config)
//	if err != nil {
//		t.Fatalf("创建音频设备失败: %v", err)
//	}
//	defer device.Close()
//
//	// 开始录制
//	err = device.StartRecording()
//	if err != nil {
//		t.Fatalf("开始录制失败: %v", err)
//	}
//
//	if !device.IsRecording() {
//		t.Error("设备应该处于录制状态")
//	}
//
//	// 录制一小段时间
//	time.Sleep(100 * time.Millisecond)
//
//	// 停止录制
//	err = device.StopRecording()
//	if err != nil {
//		t.Fatalf("停止录制失败: %v", err)
//	}
//
//	if device.IsRecording() {
//		t.Error("设备不应该处于录制状态")
//	}
//
//	// 检查是否有录制数据
//	size := device.GetCapturedDataSize()
//	if size == 0 {
//		t.Error("应该有一些录制数据")
//	}
//}
//
//func TestAudioDevicePlayback(t *testing.T) {
//	config := DefaultDeviceConfig()
//	config.LogCallback = func(message string) {}
//
//	device, err := NewAudioDevice(config)
//	if err != nil {
//		t.Fatalf("创建音频设备失败: %v", err)
//	}
//	defer device.Close()
//
//	// 先录制一些数据
//	err = device.StartRecording()
//	if err != nil {
//		t.Fatalf("开始录制失败: %v", err)
//	}
//
//	time.Sleep(100 * time.Millisecond)
//
//	err = device.StopRecording()
//	if err != nil {
//		t.Fatalf("停止录制失败: %v", err)
//	}
//
//	// 播放录制的数据
//	err = device.StartPlayback()
//	if err != nil {
//		t.Fatalf("开始播放失败: %v", err)
//	}
//
//	if !device.IsPlaying() {
//		t.Error("设备应该处于播放状态")
//	}
//
//	time.Sleep(100 * time.Millisecond)
//
//	// 停止播放
//	err = device.StopPlayback()
//	if err != nil {
//		t.Fatalf("停止播放失败: %v", err)
//	}
//
//	if device.IsPlaying() {
//		t.Error("设备不应该处于播放状态")
//	}
//}
//
//func TestAudioDevicePlaybackFromData(t *testing.T) {
//	config := DefaultDeviceConfig()
//	config.LogCallback = func(message string) {}
//
//	device, err := NewAudioDevice(config)
//	if err != nil {
//		t.Fatalf("创建音频设备失败: %v", err)
//	}
//	defer device.Close()
//
//	// 创建一些测试音频数据（静音）
//	// 对于 16-bit 单声道，每个样本 2 字节
//	// 100ms 的音频数据：44100 * 0.1 * 1 * 2 = 8820 字节
//	testData := make([]byte, 8820)
//
//	err = device.StartPlaybackFromData(testData)
//	if err != nil {
//		t.Fatalf("从数据播放失败: %v", err)
//	}
//
//	if !device.IsPlaying() {
//		t.Error("设备应该处于播放状态")
//	}
//
//	time.Sleep(50 * time.Millisecond)
//
//	err = device.StopPlayback()
//	if err != nil {
//		t.Fatalf("停止播放失败: %v", err)
//	}
//}
//
//func TestAudioDeviceGetCapturedData(t *testing.T) {
//	config := DefaultDeviceConfig()
//	config.LogCallback = func(message string) {}
//
//	device, err := NewAudioDevice(config)
//	if err != nil {
//		t.Fatalf("创建音频设备失败: %v", err)
//	}
//	defer device.Close()
//
//	// 初始状态应该没有数据
//	data := device.GetCapturedData()
//	if len(data) != 0 {
//		t.Errorf("初始状态应该没有录制数据，实际有 %d 字节", len(data))
//	}
//
//	// 录制一些数据
//	err = device.StartRecording()
//	if err != nil {
//		t.Fatalf("开始录制失败: %v", err)
//	}
//
//	time.Sleep(100 * time.Millisecond)
//
//	err = device.StopRecording()
//	if err != nil {
//		t.Fatalf("停止录制失败: %v", err)
//	}
//
//	// 应该有一些数据
//	data = device.GetCapturedData()
//	if len(data) == 0 {
//		t.Error("应该有一些录制数据")
//	}
//
//	// 清空数据
//	device.ClearCapturedData()
//	data = device.GetCapturedData()
//	if len(data) != 0 {
//		t.Error("清空后应该没有数据")
//	}
//}
//
//func TestAudioDeviceClose(t *testing.T) {
//	config := DefaultDeviceConfig()
//	config.LogCallback = func(message string) {}
//
//	device, err := NewAudioDevice(config)
//	if err != nil {
//		t.Fatalf("创建音频设备失败: %v", err)
//	}
//
//	err = device.Close()
//	if err != nil {
//		t.Fatalf("关闭设备失败: %v", err)
//	}
//
//	// 关闭后不应该处于录制或播放状态
//	if device.IsRecording() {
//		t.Error("关闭后不应该处于录制状态")
//	}
//	if device.IsPlaying() {
//		t.Error("关闭后不应该处于播放状态")
//	}
//}
