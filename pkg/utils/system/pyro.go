package system

import (
	"runtime"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/grafana/pyroscope-go"
)

func StartPyroScope() error {
	pyroscopeUrl := utils.GetEnv("PYROSCOPE_URL")
	if pyroscopeUrl == "" {
		return nil
	}

	pyroscopeAppName := utils.GetEnv("PYROSCOPE_APP_NAME")
	if pyroscopeAppName == "" {
		pyroscopeAppName = "ling-voice"
	}
	pyroscopeBasicAuthUser := utils.GetEnv("PYROSCOPE_BASIC_AUTH_USER")
	pyroscopeBasicAuthPassword := utils.GetEnv("PYROSCOPE_BASIC_AUTH_PASSWORD")
	pyroscopeHostname := utils.GetEnv("HOSTNAME")
	if pyroscopeHostname == "" {
		pyroscopeHostname = "ling-voice"
	}

	mutexRate := utils.GetIntEnv("PYROSCOPE_MUTEX_RATE")
	if mutexRate == 0 {
		mutexRate = 5
	}
	blockRate := utils.GetIntEnv("PYROSCOPE_BLOCK_RATE")
	if blockRate == 0 {
		blockRate = 5
	}

	runtime.SetMutexProfileFraction(int(mutexRate))
	runtime.SetBlockProfileRate(int(blockRate))

	_, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: pyroscopeAppName,

		ServerAddress:     pyroscopeUrl,
		BasicAuthUser:     pyroscopeBasicAuthUser,
		BasicAuthPassword: pyroscopeBasicAuthPassword,

		Logger: nil,

		Tags: map[string]string{"hostname": pyroscopeHostname},

		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,

			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})
	if err != nil {
		return err
	}
	return nil
}
