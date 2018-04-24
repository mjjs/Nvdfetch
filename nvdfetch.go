package main

import (
	"encoding/json"
	"flag"
	"fmt"
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
	} else {
		return config
	}
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

// getDownloadUrl crawls the Nvidia's webpage and parses the required webpages to find the download link for the driver
func getDownloadUrl(psId, pfId, osId int) string {
	fmt.Print("Fetching the driver download page\n\n")
	const nvidiaDownloadPage string = "https://uk.download.nvidia.com"
	const nvidiaSearchPage string = "https://www.nvidia.co.uk/Download/processDriver.aspx"

	// Generate the initial URL which redirects to the driver's download page
	driverSearchUrl := nvidiaSearchPage + "?&psid=" + strconv.Itoa(psId) + "&pfid=" + strconv.Itoa(pfId) + "&rpf=1&osid=" + strconv.Itoa(osId) + "&lid=2&lang=en-uk&ctk=0"

	// Get the driver page behind the generated driver url
	resp, err := http.Get(driverSearchUrl)
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
	downloadUrl := nvidiaDownloadPage + driverLinkRegexp.FindString(string(dlPage))

	return strings.Split(downloadUrl, "&lang")[0]
}

// parseWindowsVersion queries the Windows registry for the version number of Windows
func parseWindowsVersion() int {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	checkError(err)
	defer k.Close()

	_, _, err = k.GetStringValue("CurrentMajorVersionNumber")
	if err == nil {
		// Only Windows 10 has CurrentMajorVersionNumber
		return 10
	} else {
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

func main() {
	var osId, psId, pfId int
	// Initialize the nvml library so we can query the GPU for information
	nvml, err := nvml.New("")
	checkError(err)
	nvml.Init()
	defer nvml.Shutdown()

	firstRun := flag.Bool("f", false, "Run the first time setup and exit")
	getCurrentDriverVersion := flag.Bool("dv", false, "Print the current version of the GPU driver and exit")
	automaticMode := flag.Bool("a", false, "Automatically query the host system for required information to get the latest driver")
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
		osId = getOsId(winVer, is64())
		psId, pfId = getGpuIds(isFermi, isNotebook)

		fmt.Println("Windows version:", winVer)
		fmt.Println("Gpu model:", gpuName)
	} else {
		config := loadConfig()
		// Map config JSON to cfg struct
		var cfg cfg
		err := json.Unmarshal(config, &cfg)
		checkError(err)

		// osId = os version, psId = gpu series, pfId = gpu model
		osId = getOsId(cfg.Winver, cfg.Sixtyfour)
		psId, pfId = getGpuIds(cfg.Fermi, cfg.Notebook)
	}

	downloadUrl := getDownloadUrl(psId, pfId, osId)

	// Get current driver version and compare it to the newest
	currentVersion, err := nvml.SystemGetDriverVersion()
	checkError(err)
	currentVersionFloat, err := strconv.ParseFloat(currentVersion, 64)
	checkError(err)

	versionRegexp := regexp.MustCompile(`\d+\.\d+`)
	newestVersion, err := strconv.ParseFloat(versionRegexp.FindString(downloadUrl), 64)

	if currentVersionFloat < newestVersion {
		fmt.Println("Current version", currentVersionFloat, "<<<", newestVersion, "Newest version")
		fmt.Println(downloadUrl)
	} else {
		fmt.Println("You already have the newest driver version installed:", currentVersion)
	}
}
