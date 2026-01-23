# Rook Cartridge Writer Assistant, Implementation Plan 13: embedding the web ui

The software we're going to build here will be run on a custom retro gaming console which utilizes cartridges.
It is built on Raspberry PI CM4 Modules and the cartridges are essentially SD Card slots. 
It will be put into a cartridge which copies itself into RAM. Afterwards the user can install a cartridge he wants to change the contents of.
More details will be added lateron.

This implementation plan has a heritage; all prior ones are considered common knowldege.
It is important to keep one thing in mind when writing code: its not for the machines, its for humans. Keep that in mind, and thus refrain using single-character variable names. Instead come up with useful names for the variables.

in the previous plan we implemented a web ui which interacts with the web api. in this step we embed all the files into the go app and modify the webserver to serve all the files relative to /. 

When the user visits the embedded webserver without any local path. e.g. http://192.168.0.1/, it should be presented with the index.html from the web ui.

This implementation plan has only two tasks:

* embed everything in internal/assets/web
* provide everything to the web server to be served to clients
