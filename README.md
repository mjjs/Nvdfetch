# Nvdfetch
Nvdfetch is a CLI application written in Go for checking for new drivers for your Nvidia GPU.

There are two ways of running Nvdfetch: 
1. Run the application with no command line flags. On initial launch, the application asks the user a series of questions about their system, which help determine the correct driver to get. The answers are parsed and saved into a config file so they don't have to be entered every time.
1. Use the ```-a``` flag. This runs the application in automatic mode, which queries the host system for the required information.

If you want to run the first time setup again, use the ```-f``` flag.

# Requirements
Windows 7, 8.* or 10

## Building
Because the NVML library uses CGO, it needs to be compiled using GCC. The go tool should handle everything for you, but you need GCC to be installed and added to PATH before building Nvdfetch.

# Todo
* Add ability to download a newer driver (manually or automatically) instead of just printing the download URL
* Support for other operating systems(?)
