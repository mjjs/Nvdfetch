// TODO:
// * Find out users OS, Arch, GPU series and GPU model [✓]
// * Save info to config file [✓]
// * Map values to nvidia.com id values [✓]
// * Make a request to nvidia server to find newest driver for given parameters [✓]

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

// Check whether the system is 64bit or 32bit
func is64() bool {
	// Works only with Windows
	if _, y := os.LookupEnv("ProgramFiles(x86)"); y == true {
		return true
	} else {
		return false
	}
}

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

func main() {
	firstRun := flag.Bool("n", false, "Run the first time setup again")
	flag.Parse()
	if *firstRun {
		createConfig()
	}
	configFile := loadConfig()

	// Map config JSON to cfg struct
	var cfg cfg
	err := json.Unmarshal(configFile, &cfg)
	checkError(err)

	nvml, err := nvml.New("")
	checkError(err)
	nvml.Init()
	defer nvml.Shutdown()

	// psid = gpu series
	// pfid = gpu model
	// osid = os version
	osId := getOsId(cfg.Winver, cfg.Sixtyfour)
	psId, pfId := getGpuIds(cfg.Fermi, cfg.Notebook)
	var downloadUrl string = "http://www.nvidia.co.uk/Download/processDriver.aspx?psid=" + strconv.Itoa(psId) + "&pfid=" + strconv.Itoa(pfId) + "&rpf=1&osid=" + strconv.Itoa(osId) + "&lid=2&lang=en-uk&ctk=0"

	// Get the driver page behind the generated driver url
	resp, err := http.Get(downloadUrl)
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
	var driverLink = regexp.MustCompile(`\/Windows.*exe&lang=\w+`)
	exeLink := "http://www.nvidia.co.uk" + driverLink.FindString(string(dlPage))
	fmt.Println(exeLink)
}
