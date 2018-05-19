# Nvdfetch
Nvdfetch is a CLI application written in Go for checking for new drivers for your Nvidia GPU.

Running Nvdfetch:
* Running Nvdfetch with no command line arguments makes the application run in automatic mode, which queries the host system for the required information to find the correct drivers.
* Using the `-m` flag starts the application in manual mode. On initial launch, the user is asked a series of questions about the host system. The answers are used to determine the correct driver to get. A config file `config.json` is created based on the answers and it will be used for sequential runs.

Other good to know arguments:
* `-f` runs the first time setup again. Re-writes the config file.
* `-d` downloads the driver to disk if a newer version is found

## Requirements
Windows 7, 8.* or 10

## Installing
The latest version can be found on the [releases page of this repository](https://github.com/mjjs/Nvdfetch/releases/latest). The application does not really need to install anything, but it is good to keep in mind that the config file and downloaded driver files are saved in the same directory where the executable is being run in.

## Building from source
Because the NVML library uses CGO, it needs to be compiled using GCC. The Go tool should handle everything for you, but you need GCC to be installed and added to PATH before building Nvdfetch.

## Todo
* Add simple GUI
* Re-implement config system(?)
* Support for other operating systems(?)
