# Rook Cartridge Writer Assistant, Implementation Plan 16: api changes

The software we're going to build here will be run on a custom retro gaming console which utilizes cartridges.
It is built on Raspberry PI CM4 Modules and the cartridges are essentially SD Card slots.
It will be put into a cartridge which copies itself into RAM. Afterwards the user can install a cartridge he wants to change the contents of.
More details will be added lateron.

This implementation plan has a heritage; all prior ones are considered common knowldege.
It is important to keep one thing in mind when writing code: its not for the machines, its for humans. Keep that in mind, and thus refrain using single-character variable names. Instead come up with useful names for the variables.

## goal

The frontend Team, which works on the webapp from now on, has implemented a first change of the webapp and as a result API changes are nessecary to optimize the number of requests done by the web frontend. This implementation plan is describing what has to change and how.

The requirements are as follows: the web frontend needs to know how many of the game systems supported by a retropie cartridge are empty and how many files are present for the game systems that have files.

## implementation steps

### step one: revise api spec

First of all we have to update the api spec to reflect this change. There is one API call that needs to provide more information: /cartridgeinfo. currently, the systems list is a string array. the systems property needs to provide an array of objects. for each object two properties need to be present: system (the old string value) and filecount.
In addition to the systems array, wich should only contain systems with present files, an additional property needs to be introduced: empty_systems, which is an array of strings and contains only those game systems on the cartridge, that are empty.

Update the api spec to reflect these changes.

### step two: update the simulator

as a first step to test the api, update the simulator to reply according to the just updated spec from the previous step.

### step three: prepare the main application

the main application stores the data for the cartridgeinfo request in memory (see internal/state/cartridge_info).
that structure needs to be updated to be able to store the data needed by the web application. To ensure that the application continues to compile, put all systems found during cartridge detection (see internal/cartridge/detect.go) into the empty systems array, as that has the same structure as before.

### step four: collect the data

As the internal data structures are now on par with the new api, update the cartridge detection logic to collect all data needed. That should be done in cartridge/detect.go

### step five: update the api implementation of /cartridgeinfo

Now everything should be in place to finally implement the real cartridgeinfo api call changes. So thats what should be done now.
