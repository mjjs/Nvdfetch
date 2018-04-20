package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/mxpv/nvml-go"
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
	fmt.Println("Running first time setup. Answer these questions to determine the right drivers for you:\n")

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
func getDownloadUrl(url string) string {
	const nvidiaDownloadPage string = "http://uk.download.nvidia.com"

	// Get the driver page behind the generated driver url
	resp, err := http.Get(url)
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
	var driverLinkRegexp = regexp.MustCompile(`\/Windows.*exe&lang=\w+`)
	downloadUrl := nvidiaDownloadPage + driverLinkRegexp.FindString(string(dlPage))

	return downloadUrl
}

func main() {
	firstRun := flag.Bool("f", false, "Run the first time setup and exit")
	getCurrentDriverVersion := flag.Bool("v", false, "Print the current version of the GPU driver and exit")
	flag.Parse()

	if *firstRun {
		createConfig()
		os.Exit(0)
	}
	config := loadConfig()

	// Map config JSON to cfg struct
	var cfg cfg
	err := json.Unmarshal(config, &cfg)
	checkError(err)

	// Initialize the nvml library so we can query the GPU for information
	nvml, err := nvml.New("")
	checkError(err)
	nvml.Init()
	defer nvml.Shutdown()

	if *getCurrentDriverVersion {
		currentVersion, err := nvml.SystemGetDriverVersion()
		nvml.Shutdown()
		checkError(err)
		fmt.Print(currentVersion)
		os.Exit(0)
	}

	// osId = os version, psId = gpu series, pfId = gpu model
	osId := getOsId(cfg.Winver, cfg.Sixtyfour)
	psId, pfId := getGpuIds(cfg.Fermi, cfg.Notebook)

	// Generate the initial URL which redirects to the driver's download page
	downloadUrl := "http://www.nvidia.co.uk/Download/processDriver.aspx?psid=" + strconv.Itoa(psId) + "&pfid=" + strconv.Itoa(pfId) + "&rpf=1&osid=" + strconv.Itoa(osId) + "&lid=2&lang=en-uk&ctk=0"
	downloadUrl = getDownloadUrl(downloadUrl)

	// Get current driver version and compare it to the newest
	currentVersion, err := nvml.SystemGetDriverVersion()
	checkError(err)

	currentVersionFloat, err := strconv.ParseFloat(currentVersion, 64)
	checkError(err)

	versionRegexp := regexp.MustCompile(`\d+\.\d+`)
	newestVersion, err := strconv.ParseFloat(versionRegexp.FindString(downloadUrl), 64)

	if currentVersionFloat < newestVersion {
		fmt.Println("Current version", currentVersionFloat, "---", newestVersion, "Newest version")
		fmt.Println(downloadUrl)
	}
}
