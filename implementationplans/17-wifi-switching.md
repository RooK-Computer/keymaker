# Rook Cartridge Writer Assistant, Implementation Plan 17: wifi switching

The software we're going to build here will be run on a custom retro gaming console which utilizes cartridges.
It is built on Raspberry PI CM4 Modules and the cartridges are essentially SD Card slots.
It will be put into a cartridge which copies itself into RAM. Afterwards the user can install a cartridge he wants to change the contents of.
More details will be added lateron.

This implementation plan has a heritage; all prior ones are considered common knowldege.
It is important to keep one thing in mind when writing code: its not for the machines, its for humans. Keep that in mind, and thus refrain using single-character variable names. Instead come up with useful names for the variables.

## goal

Currently, the software configures the wifi hardware to run in Access point mode. Once the user has conected to it using the web UI built by the frontend Team, it should be possible for the user to let the console join another wifi network.

Therefore the API spec needs an update, which we then have to "fake" for the Frontend Team to build the UI against in the testing environment. Finally, in the last step, we have to really implement the logic and bring the feature up.

The basic principle is as follows: wifi.sh has a mode where it continously searches for wifi networks nearby and echos those one by one each in a single line. So in the final installment the app has to start wifi.sh and continously catch the output of it and put it into a list of visible wifi networks.

The API used by the frontend team needs a new endpoint which returns the current list of known wifi networks (which are cached by the described logic above). The Frontend team can then present all those networks to the user to select one.

Another API endpoint is needed, which should be a POST and accept two values: the name of a wifi network and the password. With that wifi.sh should be executed again to join the network. wifi.sh is already capable of joining networks so that's already done.

For testing purposes the simulator we already have should implement this post request differently: it should stop the http server for 15 seconds so that the web UI can register the disconnect which will happen on a network switch. after the 15 seconds the http component should come back online which would trigger a reconnect in the web ui (there is a timer for that) so that the frontend can update the data. That's as near as we can get to a real-world simulation. And that is enoug for our testing purposes.

## implementation steps

### step one: update api spec

The first step is rather simple: update the API spec to provide the two new endpoints. also update the docs where nessecary.

### step two: update core components

As the simulator and the main application share a huge set of components, these need to be updated with stub functions to allow the new api endpoints to exist.

### step three: update the simulator

In this step, the simulator should be updated to support the two new endpoints as described in this document.
Updating the api validator might be needed as well. For the list of Wifi Networks simply add three funny german wifi network names. References to Bielefeld and Cologne are welcome.

### step four: add the wifi surveillance logic to the main application

As the main application is more complex, updating it is split up into multiple parts: the first one being the "backoffice". Add the logic to run wifi.sh in surveillance mode in the background and collect its output. Store it in an array of strings as needed for the API. ensure it will be killed once the main process exists. As these infos are collected asynchronously, build a new component, like wifi_config.go to store the found networks. 
wifi.sh might report the same network name multiple times. always keep the list tidy and sorted.

### step five: implement the wifi fetch endpoint to the main application

Now as the data are present replace the stub implementation of the wifi list api endpoint with the collected data.

### step six: implement switch wifi network endpoint to main application

As the final step, add the logic to run `wifi.sh join` with the needed parameters to really switch the network.
Ensure that all data about the network connection are updated and presented on the screen (this should already work).
