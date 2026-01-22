# Rook Cartridge Writer Assistant, Implementation Plan 10: the main screen

The software we're going to build here will be run on a custom retro gaming console which utilizes cartridges.
It is built on Raspberry PI CM4 Modules and the cartridges are essentially SD Card slots. 
It will be put into a cartridge which copies itself into RAM. Afterwards the user can install a cartridge he wants to change the contents of.
More details will be added lateron.

This implementation plan has a heritage; all prior ones are considered common knowldege.
It is important to keep one thing in mind when writing code: its not for the machines, its for humans. Keep that in mind, and thus refrain using single-character variable names. Instead come up with useful names for the variables.
This plan covers the most important and most complex screen of the application, the main screen. A skeleton of that screen is already present.

This screen serves as the primary information to the user and as such has to display a lot of information.
For this reason, the logo will be shown on the top left of the screen to have more space for the rest of the information that need to be shown.

The screen will in its final form (for now):

* show the SSID of the Wifi the System is connected to.
* If the hotspot mode is active, will show a QR code to ease joining the access point
* display information about the cartridge inserted.
* show the IP address the machine is reachable via. Wifi takes precedence here.
* show a QR code below the IP adres which contains the hyperlinkt http://$IP_OF_THE_SYSTEM

the information can be clustered as follows:

* wifi connection state and possibly qr code
* ip reachability and qr code
* info about the cartridge

Those three clusters together with the logo allow us to segment the screen into four parts:

* top left: logo
* top right: WIFI info
* bottom left: IP info
* bottom right: cartride info

as this is a more complex endeavor, this implementation plan is split up into multiple steps:

## step one: wiring the main screen up

currently, the wifi setup screen exits the app. This needs to be changed to a transition to the main screen.

## step two: integrate libraries to generate QR codes

As we need to generate two different QR codes, we need a library to generate those. That is going to be implemented next.

## step four: preparations for the screen rendering

We don't have any layouting engine yet - while this screen is still possible without one, its a good idea to think about building a small layout engine and use that here. This part is not yet fleshed out and is up for discussion.

## step five: collecting information

Information like the IP Address are not available through central configuration components. The screen has to rely on netinfo.sh to collect those info. That script has to be called regularly - once every 30 seconds and the results need to be stored for the next 30 seconds.

## step six: show the wifi info

implement the logic to show the SSID connected to and the QR code for hotspot mode.

## step seven: show ip info

implement the logic to show the IP address (wifi takes precedence over ethernet) and the qr code with the URL.

## step eight: show info about the cartridge

use the central CartridgeInfo component to show the details about the cartridge.