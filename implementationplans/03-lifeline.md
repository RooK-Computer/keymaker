# Rook Cartridge Writer Assistant, Implementation Plan 3: lifeline

The software we're going to build here will be run on a custom retro gaming console which utilizes cartridges.
It is built on Raspberry PI CM4 Modules and the cartridges are essentially SD Card slots. 
It will be put into a cartridge which copies itself into RAM. Afterwards the user can install a cartridge he wants to change the contents of.
More details will be added lateron.

This implementation plan has a heritage; all prior ones are considered common knowldege.

We have to add a new script to the scripts folder. One important feature of the hardware is the ability to stay online even when the cartridge is removed. The onboard Hardware of the mainboard prevents that as the MOSFET controlling the ground line disconnects if the cartridge is removed. Therefore an additional circuit is present, which can override the onboard power MOSFET. that additional circuit is called lifeline.

To activate it, the GPIO pin 7 has to be driven as an output and into the LOW state. Once the lifeline is no longer needed it can be disabled again by switching pin 7 into a HIGH state. That is what we need to implement: a script, called lifeline.sh, which accepts one argument: on or off. When called with on, it should enable the lifeline circuit and exit. when called with off, it should disable it and exit.
As all scripts in the scripts folder: they will be called with sudo, so permissions are not a problem.

It is important that the script configures the pin correctly and exits - it should not block in any way. As the operating system is stripped down, not all cli tools are available. gpioset is, however.