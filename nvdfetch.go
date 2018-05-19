package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/mxpv/nvml-go"
	"golang.org/x/sys/windows/registry"
)

// Struct to hold the required information to determine which driver to download
type cfg struct {
	Winver    int  `json:"Winver,string"`
	Fermi     bool `json:"Fermi,string"`
	Notebook  bool `json:"Notebook,string"`
	Sixtyfour bool `json:"Sixtyfour,string"`
}

// Helper function to reduce LOC when checking errors
func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

// Check whether Windows is 64bit or 32bit
func is64() bool {
	// Works only with Windows
	_, y := os.LookupEnv("ProgramFiles(x86)")
	return y
}

// loadConfig reads the config file and returns its contents. If the file does not exist, createConfig will be called to create the file.
func loadConfig() []byte {
	config, err := ioutil.ReadFile("config.json")
	if os.IsNotExist(err) {
		fmt.Println("Config file not found. Generating it now. . .")
		createConfig()
		config, _ := ioutil.ReadFile("config.json")
		return config
	}
	return config
}

// createConfig asks the user for a series of questions and saves the answers into the config file.
func createConfig() {
	config, err := os.Create("config.json")
	checkError(err)
	fmt.Print("Running first time setup. Answer these questions to determine the right drivers for you:\n\n")

	var osVersion string
	for osVersion != "1" && osVersion != "2" {
		fmt.Println("Is your Windows version: \n1. Windows 7, 8 or 8.1\n2. Windows 10")
		fmt.Scanln(&osVersion)
	}

	fmt.Println("\nIs your GPU at least Fermi or newer? (400-TITAN series) y/n?")
	var fermi string
	for fermi != "y" && fermi != "n" {
		fmt.Println("Please enter either y or n")
		fmt.Scanln(&fermi)
	}

	fmt.Println("\nAre you using a notebook? y/n")
	var noteBook string
	for noteBook != "y" && noteBook != "n" {
		fmt.Println("Please enter either y or n")
		fmt.Scanln(&noteBook)
	}

	if osVersion == "1" {
		config.WriteString(`{"Winver": "7",` + "\n")
	} else {
		config.WriteString(`{"Winver": "10",` + "\n")
	}

	if fermi == "y" {
		config.WriteString(`"Fermi": "true",` + "\n")
	} else {
		config.WriteString(`"Fermi": "false",` + "\n")
	}

	if noteBook == "y" {
		config.WriteString(`"Notebook": "true",` + "\n")
	} else {
		config.WriteString(`"Notebook": "false",` + "\n")
	}

	if is64() {
		config.WriteString(`"Sixtyfour": "true"}` + "\n")
	} else {
		config.WriteString(`"Sixtyfour": "false"}` + "\n")
	}

	config.Sync()
	config.Close()
}

// getDownloadURL crawls the Nvidia's webpage and parses the required webpages to find the download link for the driver
func getDownloadURL(psID, pfID, osID int) string {
	fmt.Print("Fetching the driver download page\n\n")
	const nvidiaDownloadPage string = "https://uk.download.nvidia.com"
	const nvidiaSearchPage string = "https://www.nvidia.co.uk/Download/processDriver.aspx"

	// Generate the initial URL which redirects to the driver's download page
	driverSearchURL := nvidiaSearchPage + "?&psid=" + strconv.Itoa(psID) + "&pfid=" + strconv.Itoa(pfID) + "&rpf=1&osid=" + strconv.Itoa(osID) + "&lid=2&lang=en-uk&ctk=0"

	// Get the driver page behind the generated driver url
	resp, err := http.Get(driverSearchURL)
	checkError(err)
	defer resp.Body.Close()
	driverPage, err := ioutil.ReadAll(resp.Body)
	checkError(err)

	// Get url for the download page
	resp, err = http.Get(string(driverPage))
	checkError(err)
	dlPage, err := ioutil.ReadAll(resp.Body)
	checkError(err)

	// Parse the driver executable link from the driver page
	driverLinkRegexp := regexp.MustCompile(`\/Windows.*exe&lang=\w+`)
	downloadURL := nvidiaDownloadPage + driverLinkRegexp.FindString(string(dlPage))

	return strings.Split(downloadURL, "&lang")[0]
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

func progressBar(percentage float64) string {
	prog := ""
	for i := 0; i < 10; i++ {
		if int(percentage)/10 < i {
			s := []string{prog, "-"}
			prog = strings.Join(s, "")
		} else {
			s := []string{prog, "#"}
			prog = strings.Join(s, "")
		}
	}
	return strings.Join([]string{"[", prog, "]"}, "")
}

// Print the download progress to the console
func showProgress(remoteFileSize int64, localFile *os.File) {
	var localFileSize int64

	for remoteFileSize > localFileSize {
		localFileInfo, err := localFile.Stat()
		checkError(err)

		localFileSize = localFileInfo.Size()
		progressBar := progressBar(float64(localFileSize) / float64(remoteFileSize) * 100)

		fmt.Printf("\r%.0f%s %s %d/%dMB", float64(localFileSize)/float64(remoteFileSize)*100, "%", progressBar, localFileSize/1000000, remoteFileSize/1000000)
	}
}

func main() {
	var osID, psID, pfID int
	// Initialize the nvml library so we can query the GPU for information
	nvml, err := nvml.New("")
	checkError(err)
	nvml.Init()
	defer nvml.Shutdown()

	firstRun := flag.Bool("f", false, "Run the first time setup and exit")
	getCurrentDriverVersion := flag.Bool("dv", false, "Print the current version of the GPU driver and exit")
	automaticMode := flag.Bool("a", true, "Automatically query the host system for required information to get the latest driver")
	manualMode := flag.Bool("m", false, "Use the config file to determine the system information")
	downloadDriver := flag.Bool("d", false, "Download the driver if a newer one is found")
	flag.Parse()

	if *firstRun {
		createConfig()
		os.Exit(0)
	}

	if *getCurrentDriverVersion {
		currentVersion, err := nvml.SystemGetDriverVersion()
		checkError(err)
		nvml.Shutdown()
		fmt.Print(currentVersion)
		os.Exit(0)
	}

	if *automaticMode {
		fmt.Print("Querying system for required information\n\n")
		if runtime.GOOS != "windows" {
			fmt.Println("Unsupported operating system detected")
			os.Exit(-1)
		}
		gpuName, isNotebook, isFermi := parseGpuInfo(*nvml)
		winVer := parseWindowsVersion()
		osID = getOsID(winVer, is64())
		psID, pfID = getGpuIds(isFermi, isNotebook)

		fmt.Println("Windows version:", winVer)
		fmt.Println("Gpu model:", gpuName)
	}

	if *manualMode {
		config := loadConfig()
		// Map config JSON to cfg struct
		var cfg cfg
		err := json.Unmarshal(config, &cfg)
		checkError(err)

		// osId = os version, psId = gpu series, pfId = gpu model
		osID = getOsID(cfg.Winver, cfg.Sixtyfour)
		psID, pfID = getGpuIds(cfg.Fermi, cfg.Notebook)
	}

	downloadURL := getDownloadURL(psID, pfID, osID)

	// Get current driver version and compare it to the newest
	currentVersion, err := nvml.SystemGetDriverVersion()
	checkError(err)
	currentVersionFloat, err := strconv.ParseFloat(currentVersion, 64)
	checkError(err)

	versionRegexp := regexp.MustCompile(`\d+\.\d+`)
	newestVersion, err := strconv.ParseFloat(versionRegexp.FindString(downloadURL), 64)

	if currentVersionFloat < newestVersion {
		fmt.Println("Current version", currentVersionFloat, "<<<", newestVersion, "Newest version")

		if *downloadDriver {
			// Parse the filename out of the download URL
			filenameRegexp := regexp.MustCompile(`\d+\.\d+-.+exe`)
			filename := filenameRegexp.FindString(downloadURL)

			// Create the driver file to disk
			driverFile, err := os.Create(filename)
			checkError(err)
			defer driverFile.Close()

			// Get the data
			resp, err := http.Get(downloadURL)
			checkError(err)
			defer resp.Body.Close()

			// Get the file size of the driver
			remoteFileSize, err := strconv.ParseInt((resp.Header.Get("Content-Length")), 0, 64)
			checkError(err)

			// Show progress of the download
			go showProgress(remoteFileSize, driverFile)

			fmt.Println("Downloading file:", filename)
			// Write data to file
			_, err = io.Copy(driverFile, resp.Body)
			checkError(err)
		} else {
			fmt.Println(downloadURL)
		}

	} else {
		fmt.Println("You already have the newest driver version installed:", currentVersion)
	}
}
