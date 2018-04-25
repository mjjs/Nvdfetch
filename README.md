# Nvdfetch
Nvdfetch is a CLI application written in Go for checking for new drivers for your Nvidia GPU.

There are two ways of running Nvdfetch: 
1. Run the application with no command line arguments. This makes the application run in automatic mode, which queries the host system for the required information to find the correct drivers.
1. Run the application with the ```-m``` argument. On initial launch, the application asks a series of questions about the host system. The answers are used to determine the correct driver to get. A config file ```config.json``` is created based on the answers and it will be used the next time the program is run.

If you want to run the first time setup again, use the ```-f``` flag.

# Requirements
Windows 7, 8.* or 10

## Building
Because the NVML library uses CGO, it needs to be compiled using GCC. The go tool should handle everything for you, but you need GCC to be installed and added to PATH before building Nvdfetch.

# Todo
* Add ability to download a newer driver (manually or automatically) instead of just printing the download URL
* Support for other operating systems(?)
