// This file is used for mapping the OS version, GPU series and GPU model into the ID values used by the Nvidia website
package main

func getOsId(windowsVersion int, sixtyFourBit bool) int {
	// Windows 7 and 8 / 8.1 can be grouped together, because they use the same drivers
	if windowsVersion == 7 || windowsVersion == 8 {
		if sixtyFourBit {
			return 19
		} else {
			return 18
		}
	} else if windowsVersion == 10 {
		if sixtyFourBit {
			return 57
		} else {
			return 56
		}
	} else {
		panic("Unsupported operating system. Either your Windows is too old (older than Windows 7) or you have errors in your config file.")
	}
}

func getGpuIds(fermi, mobile bool) (gpuSeries, gpuModel int) {
	// If 400 series or better, we can use any model up to TITAN
	if fermi {
		if mobile {
			gpuSeries = 64
			gpuModel = 637
		} else {
			gpuSeries = 85
			gpuModel = 660
		}
	} else {
		// 8, 9, 100, 200, 300 (and GeForce 405 GPU) series use the same drivers
		if mobile {
			gpuSeries = 62
			gpuModel = 460
		} else {
			gpuSeries = 52
			gpuModel = 450
		}
	}
	return
}
