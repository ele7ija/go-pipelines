# go-pipelines

This repository contains a batch image management server.

To make the server process the image as fast and efficient as possible, 
a few different types of processing were introduced: _sequential_, _concurrent_ and _different pipeline processing_.
Take a look at [`cmd/go-pipelines/main.go`](cmd/go-pipelines/main.go)

The best solution turned out to be *Bounded pipeline processing*. This solution bounds the number of goroutines depending on how complex a phase is. A more resource-consuming phase gets more goroutines. **Results are presented in an article [here](https://itnext.io/performant-image-processing-with-go-pipelines-and-bounded-concurrency-3f721ec5dde8#Results)**. 

A package which provides a framework to implement pipeline processing is here: [github.com/ele7ija/pipeline](https://github.com/ele7ija/pipeline).

The project isn't deployed anywhere, but it is dead easy to run it on your local machine, take a look at links below.

## Links

[Running the project](Running.md)

[Frontend for the server](https://github.com/ele7ija/gollery)

### Demo 

_Disclaimer_: file selection window not visible for some reason.

![Demo](assets/demo-avi-2x.gif)
