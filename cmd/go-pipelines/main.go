package main

import (
    "fmt"
    gopipelines "github.com/ele7ija/go-pipelines"
)

func DoConcurrentApi() {

    genFilter := gopipelines.NewSerialFilter(gopipelines.NewGenWorker(100))
    sqFilter1 := gopipelines.NewParallelFilter(gopipelines.NewSqWorker())
    sqFilter2 := gopipelines.NewParallelFilter(gopipelines.NewSqWorker())

    pipeline := gopipelines.NewPipeline(genFilter, sqFilter1, sqFilter2)

    sl := make([]int, 100)
    ch := make(chan interface{}, 100)
    for i := range sl {
        ch <- i
    }
    close (ch)

    for n := range pipeline.Filter(ch) {
        fmt.Print(n)
    }
}