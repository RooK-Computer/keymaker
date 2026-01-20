# Rook Cartridge Writer Assistant, Implementation Plan 6: WIFI hotspot

The software we're going to build here will be run on a custom retro gaming console which utilizes cartridges.
It is built on Raspberry PI CM4 Modules and the cartridges are essentially SD Card slots. 
It will be put into a cartridge which copies itself into RAM. Afterwards the user can install a cartridge he wants to change the contents of.
More details will be added lateron.

This implementation plan has a heritage; all prior ones are considered common knowledge.

This time we are going to implement another helper script that will be put in the scripts folder. After startup, the console should open a wifi for the user to connect to. We need to build a script, that will be used in multiple situations around networking:

* when the system boots up, a wifi hotspot needs to be configured
* once the user has connected and wants the console to join another network, switching the network has to be done by the script as well.

The script will be called "wifi.sh".

Lets discuss the details:

## hotspot mode

when being called to switch to hotspot mode, the script should interact with NetworkManager to establish an unencrypted hotspot on a private IP network. We will use 192.168.0.0/24 for now. The Network Name should be set to "RooK-" and a 4-digit random number. It should give out IP adresses to clients joining the network. 

For hotspot mode activation, a single "hotspot" argument will be passed.

The script needs to be aware wether the hotspot is already configured and can be re-enabled or wether the hotspot config is not yet present in NetworkManager and needs to be created.

## network search

While in hotspot mode, the script may be called with the argument "surveillance". It should block then and stream wifi networks that are visible - in a simple line-by-line manner. For example, if the network chip detects two networks named "AroundTheBlock" and "HomeSweetHome", the script should output:

```
AroundTheBlock
HomeSweetHome
```

the script needs to be killed in this mode without leaving the wifi environment in an unusable state.

## joining an existing network

If the user chooses to connect to a different network, the script will be called with three arguments: the first one being "join", while the second one is the name of the network to join and the third one is the password to use when connecting.

In this mode its also important to check if the network is already known to NetworkManager and update it.
While switching from hotspot to an existing network, the dhcp server is no longer needed. While writing this document it is unclear wether that will be handled by NetworkManager automatically or not.

