# Rook Cartridge Writer Assistant, Implementation Plan 8: insert cartridge screen

The software we're going to build here will be run on a custom retro gaming console which utilizes cartridges.
It is built on Raspberry PI CM4 Modules and the cartridges are essentially SD Card slots. 
It will be put into a cartridge which copies itself into RAM. Afterwards the user can install a cartridge he wants to change the contents of.
More details will be added lateron.

This implementation plan has a heritage; all prior ones are considered common knowldege.
This plan will cover the next step of the app startup: asking the user to insert the cartridge to write to.
An skeleton screen for this use case already exists (internal/app/screens/insert_cartridge.go), we will now flesh out this step to continue building the startup process.

Currently, when the application starts, it ejects the cartridge and asks the user to remove it. It uses the scripts eject_sd.sh and wait_for_sd_eject.sh. Those calls are encapsulated in a separate go object.

Once both events happened, the application exits. So far for the current setup. Here is what needs to be changed:

## step one: hand over to the insert cartridge screen

Remove the logic to exit the app from the cartridge eject screen. Instead, the cartridge eject screen should hand over to the insert cartridge screen.

## step two: implement the backing infrastructure

The app will have multiple components which will need to know things about the cartridge:

* wether it is present or not.
* wether it is currently mounted
* wether it is a retropie system and which systems are supported on that cartridge
* wether there is a cartridge present to be worked on
* wether it is busy or not

Build a singleton component which keeps these information at hand for the app. Name that Component CartridgeInfo.

## step three: flesh out the cartridge insert screen

This screen will be a bit more complex than the eject screen. It has to:

* start by telling the user to insert a cartridge (the skeleton screen does just that)
* wait for an inserted cartridge by using the wait_for_sd.sh script
* tell the user it is analyzing the cartridge
* then check wether its retropie or not by using the is_sd_retropie.sh script.
* if that succeeds use the sd_retropie_systems.sh script to collect the systems available on that cartridge
* should one of the previous steps mount the cartridge and keep it mounted, unmount it again.
* update the central cartridge information component from step two with the information just gathered.

## step four:

For now, just as with the eject screen: once we reach this point, the application should exit and return the framebuffer to a text usable mode.

## final notes

This screen interacts heavily with the scripts we had built beforehand. Not all of them are wrapped in Go Objects, this needs to be done for this implementation plan as well. it is important to keep the code style in sync with the rest of the project: refrain from single-character variable names, try finding variable and function names useful for the human reader and debugger of the code. 