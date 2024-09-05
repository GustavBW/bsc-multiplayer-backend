# Multiplayer Backend
A simple http and websocket server handling multiplayer

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