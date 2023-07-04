
## About

This is a simple api to generate random numbers, random strings and UUID's. This project was built using the GO language.

One of the features of this API is to use reddit comments as seeds, to generate what I am calling a "free-will" seed. 
The thought behind this, is that if you believe in free will, then this seed should be random and unable to predict

## Warnning
This API uses GO's random functions.
The Reddit random function does use the GO random function seeded by reddit comments.
<BR>This has not been tested for and should not be used for secure cryptographic purposes

## Optional command line arguments 

| Param | Desc | Default |
|-------|------|---------|
|addr|Set the listening ip and port for the server|4000|

You can also redirect the logs to log files by adding this to the command
```
go run ./cmd/api >>/tmp/info.log 2>>/tmp/error.log
```

## Build and run this using Podman/Docker

### Build
From the root of the project
```
sudo podman build -f ./docker/Dockerfile . -t radmonnumberapi
```

### Run
```
sudo podman run -d --rm -p 4000:4000 radmonnumberapi
```

## Dependencies

| Name | Version | Disc                    |
|------|---------|-------------------------|
|github.com/google/uuid|v1.3.0| Used to generate UUID's |