# Keymaker

Keymaker helps you manage your cartridges contents, and all you need is a capable RooK.

It allows you to:

* Copy games onto retropie setups on cartridges
* Copy games from retropie setups on cartridges
* write complete images onto cartridges

and all you need is your RooK and a device with a web browser.

## compatible RooKs

* RooK Mk3.1 + Lifeline Mod

## Components

Keymaker consists of a few moving parts:

* Shell scripts wrap the operating system operations and need to be put in $PATH.
* The core of Keymaker - an application in go, which shows some fancy UI on the Screen connected to RooK
* An HTTP API to do all the management.
* A Web App that interacts with said API

