# Rook Cartridge Writer Assistant, Implementation Plan 4: architecture update

The software we're going to build here will be run on a custom retro gaming console which utilizes cartridges.
It is built on Raspberry PI CM4 Modules and the cartridges are essentially SD Card slots. 
It will be put into a cartridge which copies itself into RAM. Afterwards the user can install a cartridge he wants to change the contents of.
More details will be added lateron.

This implementation plan has a heritage; all prior ones are considered common knowldege.

What is needed for this step is a small architecture overhaul. for now, the screen renderer simply renders whats visible on the screen, while the logic is kept in the main app object. That will not scale well with the project, as it will be getting more complex from this point on. So we have to rebuild the architecture to fit our needs:

Screen Objects represent the state of the application. As a result the logic that should be executed alongside with a screen should be tied to the screen. So we should flesh every screen out in a different file and not keep them in a single one. Screens should have a start and stop function which should be called when a screen starts and stops, respectively.

Update the screen type to reflect that; also move the screens we have out into separate files; move the logic to eject the sd card and waiting for the sd card ejection into the eject screen.

Once that's done, add the code to execute `lifeline on` to the start function of the ejection screen.
Once the cartridge is ejected, keep the current logic and the app should exit.
