package main

import (
	"os"
	"regexp"

	nvml "github.com/mxpv/nvml-go"
	"golang.org/x/sys/windows/registry"
)

// Check whether Windows is 64bit or 32bit
func is64() bool {
	// Works only with Windows
	_, y := os.LookupEnv("ProgramFiles(x86)")
	return y
}

// parseWindowsVersion queries the Windows registry for the version number of Windows
func parseWindowsVersion() int {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	checkError(err)
	defer k.Close()

	_, _, err = k.GetIntegerValue("CurrentMajorVersionNumber")
	if err == nil {
		// Only Windows 10 has CurrentMajorVersionNumber
		return 10
	}
	currentVersion, _, err := k.GetStringValue("CurrentVersion")
	checkError(err)

	switch currentVersion {
	case "6.1":
		return 7
	case "6.2":
		return 8
	case "6.3":
		return 8
	default:
		return 10
	}

}

func parseGpuInfo(nvml nvml.API) (string, bool, bool) {
	gpu, err := nvml.DeviceGetHandleByIndex(0)
	checkError(err)
	gpuName, err := nvml.DeviceGetName(gpu)
	checkError(err)

	// Check for M in the GPU name indicating a notebook model GPU
	mobileGpuRegexp := regexp.MustCompile(`(GT|GTX)\s\d+M`)
	isMobile := mobileGpuRegexp.MatchString(gpuName)

	// Check if the gpu model is a Tesla (100) series GPU or newer
	teslaRegexp := regexp.MustCompile(`(GTX|GT)\s\d+`)
	isFermi := false

	// Check if the gpu model is a Fermi (400) series GPU or newer
	if teslaRegexp.MatchString(gpuName) {
		fermiRegexp := regexp.MustCompile(`\s([456789]|1\d)\d+`)
		isFermi = fermiRegexp.MatchString(gpuName)
	}

	return gpuName, isMobile, isFermi
}
