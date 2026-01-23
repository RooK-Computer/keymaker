# Rook Cartridge Writer Assistant, Implementation Plan 12: building the web app

The software we're going to build here will be run on a custom retro gaming console which utilizes cartridges.
It is built on Raspberry PI CM4 Modules and the cartridges are essentially SD Card slots. 
It will be put into a cartridge which copies itself into RAM. Afterwards the user can install a cartridge he wants to change the contents of.
More details will be added lateron.

This implementation plan has a heritage; all prior ones are considered common knowldege.
It is important to keep one thing in mind when writing code: its not for the machines, its for humans. Keep that in mind, and thus refrain using single-character variable names. Instead come up with useful names for the variables.
We reached the stage of the project where we need to start working on the web ui. We don't have a toolchain for that yet, so that's part of this implementation plan.

We are going to build the whole web app using React, which will interact with the api we implemented earlier. 

When the user first opens the web browser and connects, he will be greeted with an overview of the cartridge from cartridge info and has up to three options:

* eject the cartridge to switch to a different one
* flash the cartridge with a new image
* start managing retropie games

The first two should be trivial to understand how the workflow will be: ejection more or less triggers a continous loop of fetching cartridge infos, while the user will be able to remove one cartridge and insert another one.

When flashing, the user has to choose an file (only .img.gz files are allowed) which will then be flashed to the system.

when using the last option, the user should get a user interface where he can see all systems of the retropie environment and list all games once a system is selected.

Deleting games, downloading games and uploading games should be possible in that ui. 

For now, we are not bothering making the UI nice - no styling in anyway. having a white background and not setting any font is totally fine for a prototype.

## step one: setting up the pipeline

First things first, we have to set up the pipeline in this project. We are going to write exclusively typescript and are going to build everything on top of react. Support for CSS modules is also important.

The source code will live in a folder called webapp. The compiled result should be put in internal/assets/web.

Building it should be incorporated into the Makefile, in a way that it will be run prior to the go compile phase.

Build the basic components in this step as well: the root react component, index.html, things like that. Everything that is needed to run the web pipeline.

## step two: implementing the api client

in the folder api-spec you find the openapi specification implemented by the server. we need some way of wrapper for these calls for easier interaction with the api. its fine to use some kind of code generator for it.

## step three: implement the main view

upon starting the session, the user sees this main view. That should in regular intervals fetch the cartridge infos and show them. And based on the results allow the user to eject, flash or manage retropie.

## step four: implement eject

The easiest of all commands is the eject button. as such we start with this one.

## step five: implement flashing

flashing the cartridge should work as described. while the flash is in progress, the progress should be visualized. For now, printing the relative progress in percent is enough, no fancy progress bars please.

## step six: managing retropie

upon pressing that button, flashing and eject should not be an option anymore. a back or close button should bring the user back to the main screen. On this step we only show the systems available and make them interactive - the user can select a system.

## step seven: list all games, download games

in this step we add showing the games of a system on the retropie cartridge to the ui. the games are listed and can be selected for downloading.

## step eight: uploading new games

in this step, we implement the upload feature. after selecting a system, an upload button will be visible, which allows the user to select a file which will then be uploaded and unzipped.

## step nine: deleting games

In this step, every game of a system will get the option to be deleted. as easy as that.

