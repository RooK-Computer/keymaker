# Rook Cartridge Writer Assistant, Implementation Plan 11: http api

The software we're going to build here will be run on a custom retro gaming console which utilizes cartridges.
It is built on Raspberry PI CM4 Modules and the cartridges are essentially SD Card slots. 
It will be put into a cartridge which copies itself into RAM. Afterwards the user can install a cartridge he wants to change the contents of.
More details will be added lateron.

This implementation plan has a heritage; all prior ones are considered common knowldege.
It is important to keep one thing in mind when writing code: its not for the machines, its for humans. Keep that in mind, and thus refrain using single-character variable names. Instead come up with useful names for the variables.
This plan covers some ground work for the upcoming features that are to be implemented. 
The application needs to expose a certain amount of features via HTTP, so we are going implement that now.

The API we have to expose covers a lot of things:

* uploading an image to be written to the cartridge in total (one could say dd over http, dd being the command line tool to write directly to block devices).
* accessing the games installed on a retropie cartridge and downloading or deleting them
* uploading new games to certain systems of a retropie cartridge.
* ejecting the cartridge to work on another one
* providing info about the current cartridge as JSON (from CartridgeInfo)

A few of these use cases need the cartridge in a specific state. For example: getting a list of games of a system on a retropie cartridge only works if the retropie detection flagged the cartridge as retropie and the cartridge is mounted so the application can access the files on it. So there is a certain need to automatically switch the state of the cartridge based on the api call coming in: when requesting a list of games of a specific system and the cartridge is not mounted: mount it automatically. If the request to overwrite the cartridge as a whole is coming in and the cartridge is mounted, unmount it prior to firing up gunzip and dd.

One thing is important to keep in mind: the application is run off a ramdisk, thus memory is a sparse resource. Its unacceptable to keep uploads from the user of the API in memory, they need to be written out to disk as fast as possible.

So lets define the API calls more specific:

* [GET] api/v1/cartridgeinfo - get cartridge info
* [POST] api/v1/flash - write an image to the cartridge directly
* [GET] api/v1/retropie/ - fetch a list of systems
* [GET] api/v1/retropie/{SYSTEM} - fetch a list of games of a system
* [GET] api/v1/retropie/{SYSTEM}/{GAME} - get one game. If its a folder, automatically zip it first.
* [DELETE] api/v1/retropie/{SYSTEM}/{GAME} - delete one game.
* [POST] api/v1/retropie/{SYSTEM}/{GAME} - upload one game. if its a zip file, decompress it.
* [POST] api/v1/eject - prepare the cartridge for ejection, switch to the ejection screen.

A few things are important: as there are already scripts and wrappers in place to mount and unmount the cartridge: the cartridge root point is /cartridge. every path from now on in this document is relative to that point. So if the plan mentions /home/pi/retropie, the real final path on the system is /cartridge/home/pi/retropie.

For retropie cartridges, there is a convention to place the games in /home/pi/RetroPie/roms/{SYSTEM}/. 
Certain systems will need more than one file to provide a game. Those will be uploaded as ZIP files and need to be unpacked. Afterwards the directory structured needs to be checked to avoid having more than the needed folders. After an upload with a ZIP file is done, the files of the game should be in /home/pi/RetroPie/roms/{SYSTEM}/{GAMENAME}. If the ZIP file has everything in one subfolder, it may happen that the GAMENAME folder contains only one subfolder. In that case all files have to be moved up one folder and the then empty subfolder needs to be deleted.

as this plan is quite complex, it is split up into multiple parts:

## step one: spec the API

First of all, specify the API for others to read. It should always everywhere rely on json for data being replied by the server. for uploads, the body should be the contents of the file to be uploaded - with one header being used to tell beforehand the size of the file being upload.
Create an OpenAPI specification for the API. Put it under a separate folder `api-spec´. 
Once the specification is written, generate html documentation for it under the same folder.

## step two: implement GET ...cartridgeinfo

As the title says, implement the cartridgeinfo GET path.
Prepare the webserver component for it and also ensure that it is started up during the launch of the app.
Keep in mind: the web server will lateron also server a web application, not just the API. Integrate the API with that in mind.

## step three: implement GET ...retropie

The title says it all. Implement fetching the systems list as json array.

## step four: implement GET ...retropie/{SYSTEM}

Again: implement the get to fetch a list of games for one system.

## step five: implement GET ...retropie/{SYSTEM}/{GAME}

implement fetching a game. If the game is a folder, zip it prior to returning it over http. Delete the zip file afterwards.

## step six: implement POST ...retropie/{SYSTEM}/{GAME}

implement uploading a game. Might be a zipfile to be unpacked. The zip file should be removed after unpacking.

## step seven: implement DELETE ...retropie/{SYSTEM}/{GAME}

this call deletes a game. the details should be clear by now.

## step eight: implement POST ...eject

this is a screen switch: the main screen has to switch over to the cartridge eject screen to eject the cartridge.

## step nine: implement POST ...flash

this is the most complex part. ensure that the cartridge is unmounted. Build a script called flash.sh, which pipes its input into gunzip and dd onto the cartridge. Follow the implementation style of the other scripts. Execute that script and pipe the output of the http request coming in directly to the script. There is not enough space in RAM or on disk to store the upload prior to writing it to the cartridge.
