# Rook Cartridge Writer Assistant, Implementation Plan 1: scripts

The software we're going to build here will be run on a custom retro gaming console which utilizes cartridges.
It is built on Raspberry PI CM4 Modules and the cartridges are essentially SD Card slots. 
It will be put into a cartridge which copies itself into RAM. Afterwards the user can install a cartridge he wants to change the contents of.
More details will be added lateron.

In this step, we will do the ground work to use in later implementation plans: the system integration.
This repository should contain a scripts folder which will have shell scripts which can be used by the final software to interact with the file systems and the environment.

The Scripts we need are:

* is_sd_present.sh should check wether the sd card is present in the system as a block device
* eject_sd.sh should unbind the driver so that the block device for an sd card vanishes. It should unmount beforehand.
* wait_for_sd_eject.sh should wait until the kernel logs the removal of the card.
* wait_for_sd.sh should wait until a new sd card is present.
* mount_sd.sh should check the partitioning of the sd card, find out the root partition and mount that as /cartridge. the /cartridge folder may not be present so it may need to be created
* is_sd_retropie.sh should check wether /cartridge/opt/retropie is present and wether /cartridge/home/pi/RetroPie/roms is present.
* sd_retropie_systems.sh should return a list of rom types supported by evaluating /cartridge/home/pi/RetroPie/roms and the subfolders present there.

All these scripts should run on Raspberry PI OS Trixie, albeit a lightweight image without many programs installed. fdisk and parted are present for partitioning analysis. They will be called using sudo. 
