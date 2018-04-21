# Nvdfetch
Nvdfetch is a CLI application written in Go for checking for new drivers for your Nvidia GPU.

On initial launch, the application asks the user a series of questions about their system, which help determine the correct driver to get. The answers are parsed and saved into a config file so they don't have to be entered every time.

Currently Nvdfetch works only on Windows 7, 8.* and 10.

## Todo
* Add automatic mode which determines the system's GPU and OS version without asking the user
* Add ability to download a newer driver (manually or automatically)
* Support for other operating systems(?)
