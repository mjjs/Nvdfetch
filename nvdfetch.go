package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/mxpv/nvml-go"
)

const nvidiaDownloadPage = "https://uk.download.nvidia.com"
const nvidiaSearchPage = "https://www.nvidia.co.uk/Download/processDriver.aspx"

// Struct to hold the required information to determine which driver to download
type cfg struct {
	Winver    int  `json:"Winver"`
	Fermi     bool `json:"Fermi"`
	Notebook  bool `json:"Notebook"`
	Sixtyfour bool `json:"64bit"`
}

type flags struct {
	firstRun       bool
	printVersion   bool
	manualMode     bool
	downloadDriver bool
}

type sysInfo struct {
	osID, gpuSeriesID, gpuModelID int
}

// Helper function to reduce LOC when checking errors
func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

// loadConfig reads the config file and returns its contents. If the file does not exist, createConfig will be called to create the file.
func loadConfig() *cfg {
	cfg := new(cfg)
	config, err := ioutil.ReadFile("config.json")
	if os.IsNotExist(err) {
		fmt.Println("Config file not found. Generating it now. . .")
		createConfig()
		config, _ = ioutil.ReadFile("config.json")
	}
	json.Unmarshal(config, &cfg)
	return cfg
}

// createConfig asks the user for a series of questions and saves the answers into the config file.
func createConfig() {
	configFile, err := os.Create("config.json")
	if err != nil {
		log.Printf("")
	}

	config := cfg{Winver: 10, Fermi: false, Notebook: false, Sixtyfour: true}
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

	var sixtyFour string
	fmt.Println("\nIs your operating system 64bit?")
	for sixtyFour != "y" && sixtyFour != "n" {
		fmt.Println("Please enter either y or n")
		fmt.Scanln(&sixtyFour)
	}

	if osVersion == "1" {
		config.Winver = 7
	}

	if fermi == "y" {
		config.Fermi = true
	}

	if noteBook == "y" {
		config.Notebook = true
	}

	if sixtyFour == "n" {
		config.Sixtyfour = false
	}

	configJSON, err := json.MarshalIndent(&config, "", "  ")
	if err != nil {
		// Handle error
	}

	configFile.Write(configJSON)
	configFile.Sync()
	configFile.Close()
}

// getDownloadURL crawls the Nvidia's webpage and parses the required webpages to find the download link for the driver
func getDownloadURL(psID, pfID, osID int) string {
	fmt.Print("Fetching the driver download page\n\n")

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

func parseFlags() *flags {
	f := new(flags)
	flag.BoolVar(&f.firstRun, "f", false, "Run the first time setup and exit")
	flag.BoolVar(&f.printVersion, "dv", false, "Print the current version of the GPU driver and exit")
	flag.BoolVar(&f.manualMode, "m", false, "Use the config file to determine the system information")
	flag.BoolVar(&f.downloadDriver, "d", false, "Download the driver if a newer one is found")
	flag.Parse()

	return f
}

func downloadDriver(url string) {
	// Parse the filename out of the download URL
	filenameRegexp := regexp.MustCompile(`\d+\.\d+-.+exe`)
	filename := filenameRegexp.FindString(url)

	// Create the driver file to disk
	driverFile, err := os.Create(filename)
	checkError(err)
	defer driverFile.Close()

	// Get the data
	resp, err := http.Get(url)
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
	if err != nil {
		log.Printf("An error occured when writing the file: %v", err)
		return
	}
}

func main() {
	sysInfo := new(sysInfo)
	args := parseFlags()

	if args.firstRun {
		createConfig()
		os.Exit(0)
	}

	// Initialize the nvml library so we can query the GPU for information
	nvml, err := nvml.New("")
	if err != nil && !args.manualMode {
		log.Fatalf("An error occurred, which is preventing automatic mode from continuing: %v", err)
	}
	nvml.Init()
	defer nvml.Shutdown()

	if args.printVersion {
		currentVersion, err := nvml.SystemGetDriverVersion()
		if err != nil {
			log.Fatal("Error querying for driver version")
		}
		nvml.Shutdown()
		fmt.Print(currentVersion)
		os.Exit(0)
	}

	if args.manualMode {
		config := loadConfig()

		sysInfo.osID = getOsID(config.Winver, config.Sixtyfour)
		sysInfo.gpuSeriesID, sysInfo.gpuModelID = getGpuIds(config.Fermi, config.Notebook)
	} else {
		fmt.Print("Querying system for required information\n\n")
		if runtime.GOOS != "windows" {
			fmt.Println("Unsupported operating system detected")
			os.Exit(-1)
		}
		gpuName, isNotebook, isFermi := parseGpuInfo(*nvml)
		winVer := parseWindowsVersion()
		sysInfo.osID = getOsID(winVer, is64())
		sysInfo.gpuSeriesID, sysInfo.gpuModelID = getGpuIds(isFermi, isNotebook)

		fmt.Println("Windows version:", winVer)
		fmt.Println("Gpu model:", gpuName)
	}

	downloadURL := getDownloadURL(sysInfo.gpuSeriesID, sysInfo.gpuModelID, sysInfo.osID)

	// Get current driver version and compare it to the newest
	currentVersion, err := nvml.SystemGetDriverVersion()
	checkError(err)
	currentVersionFloat, err := strconv.ParseFloat(currentVersion, 64)
	checkError(err)

	versionRegexp := regexp.MustCompile(`\d+\.\d+`)
	newestVersion, err := strconv.ParseFloat(versionRegexp.FindString(downloadURL), 64)
	if err != nil {
		// Handle error
	}

	if currentVersionFloat < newestVersion {
		fmt.Println("Current version", currentVersionFloat, "<<<", newestVersion, "Newest version")

		if args.downloadDriver {
			downloadDriver(downloadURL)
		} else {
			fmt.Println(downloadURL)
		}

	} else {
		fmt.Println("You already have the newest driver version installed:", currentVersion)
	}
}
