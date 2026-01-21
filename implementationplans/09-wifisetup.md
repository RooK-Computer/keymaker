# Rook Cartridge Writer Assistant, Implementation Plan 9: wifi setup screen

The software we're going to build here will be run on a custom retro gaming console which utilizes cartridges.
It is built on Raspberry PI CM4 Modules and the cartridges are essentially SD Card slots. 
It will be put into a cartridge which copies itself into RAM. Afterwards the user can install a cartridge he wants to change the contents of.
More details will be added lateron.

This implementation plan has a heritage; all prior ones are considered common knowldege.
It is important to keep one thing in mind when writing code: its not for the machines, its for humans. Keep that in mind, and thus refrain using single-character variable names. Instead come up with useful names for the variables.
In this step of the project we are going to add another screen, which covers the step of setting up the wifi connection.

For this to work, we will need another central component collecting data regarding the network configuration, just like the CartridgeInfo component. As it worked before, this is also a multi-step implementation plan.

## step one: central wifi configuration component

We have to set up a central (meaning: go singleton) component which stores the desired wifi configuration:

* wether the system should be connecting to a wifi network or host an access point
* if its connecting to a network, the name of the SSID
* the password for the network (if connecting to one is desired)

As you can see, in access point mode, only the first of the three properties are relevant as the ap mode autoconfigures itself from the scripts provided to do so.

If the application starts up the settings need to be detectable as unknown. That is important. The reason will be clear in the next step.

## step two: build a wifi configuration screen

This screen will be used to configure the wifi network according to the configuration stored in the central wifi configuration component. It does so by executing wifi.sh with the right parameters.

If the central configuration is in unknown state (like when the app starts), it should use netinfo.sh to find out wether a wifi connection is already established or wether an ethernet connection is established.

If so, it should not do anything and exit the application.
If no network connection is available, it should set up an access point and update the central configuration component to reflect that.

Again: once this screen is done, it should exit the application for now.

While all this is happening, the screen should tell the user, that the wifi network is being set up.

## step three: wire it into the cartridge insert screen

The cartridge insert screen currently exits the application. It should not do that anymore and instead switch to the wifi configuration screen.
