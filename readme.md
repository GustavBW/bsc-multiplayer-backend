# Multiplayer Backend
A simple http and websocket server handling multiplayer

## Modes
The program can be directed to read different .env files on startup using the flags:
```bash
--dev
#or
--prod
```
Which will attempt to read a "dev.env" or "prod.env" file respectively.

### Options
For development, it might be useful to switch the datatype of any broadcasted messages to Base16 (hexademical) instead of the default binary messages. 
```bash
    go run ./src messageEncoding="base16" # Default: "none"
```

## CLI Tools
This service is the single source of thruth for multiplayer event handling. Therefore some tools are provided to make it easier to port specifications to other languages and the like. 
These tools can be invoked by running the executable with the 
```bash
--tool
```
flag. This wont start the service, but rather end it when execution is complete.

### Print Event Specifications
Prints all event specifications and associated data.

Example:
```bash
go run ./src --tool --print-event-specs --output="<path>"

    path: Defaults to EventSpecifications-<program version>.ts
    Output type is derived from path.
```