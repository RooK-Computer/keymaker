# Rook Cartridge Writer Assistant, Implementation Plan 7: wifi infos

The software we're going to build here will be run on a custom retro gaming console which utilizes cartridges.
It is built on Raspberry PI CM4 Modules and the cartridges are essentially SD Card slots. 
It will be put into a cartridge which copies itself into RAM. Afterwards the user can install a cartridge he wants to change the contents of.
More details will be added lateron.

This implementation plan has a heritage; all prior ones are considered common knowldege.
It is time for another script in the scripts folder. This time we have to collect network information.

The script will be called "netinfo.sh" and will always be called with only one argument, where each argument will ask for another network related information. As NetworkManager is active, all data should be obtainable by querying Networkmanager.

## wifi-ssid

When called with `wifi-ssid` the script should interact with networkamanger to collect the SSID of the wifi network currently connected to (or currently hosting, if the hotspod made by wifi.sh is active).

If the wifi chip is not connected to any network, it should output nothing.

## wifi-ip

When called with `wifi-ip` the IPv4 Adress of the wifi chip should be shown. Again: if the chip is not connected, it shouldn't show output anything.

## ethernet-ip

When called with `ethernet-ip` the IPv4 Adress of the ethernet chip should be shown - but only if the link is up and active. Otherwise - again - nothing should be printed by the script.
