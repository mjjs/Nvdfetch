# Nvdfetch
Nvdfetch is a CLI application written in Go for checking for new drivers for your Nvidia GPU.

On initial launch, the application asks the user a series of questions about their system, which help determine the correct driver to get. The answers are parsed and saved into a config file so they don't have to be entered every time.

When run with the ```-a``` flag, the host system will be queried for the required information to determine the correct driver.

Currently Nvdfetch works only on Windows 7, 8.* and 10.

## Todo
* Add ability to download a newer driver (manually or automatically) instead of just printing the download URL
* Support for other operating systems(?)
