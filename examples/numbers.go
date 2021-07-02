package examples

import (
	"fmt"
	pipeApi "github.com/ele7ija/go-pipelines/internal"
	"time"
)

type GenWorker struct {

	max int
	curr int
}

func NewGenWorker(max int) *GenWorker {

	return &GenWorker{max: max, curr: 0}
}

func (f*GenWorker) Work(in interface{}) interface{} {

	time.Sleep(time.Millisecond * 2)
	curr := f.curr
	f.curr++
	return curr
}

type SqWorker struct {
}

func NewSqWorker() *SqWorker {

	return &SqWorker{}
}

func (f*SqWorker) Work(in interface{}) interface{} {

	time.Sleep(time.Millisecond * 2)
	n := in.(int)
	return n * n
}

func DoConcurrentApi() {

	genFilter := pipeApi.NewParallelFilter(NewGenWorker(100))
	sqFilter1 := pipeApi.NewParallelFilter(NewSqWorker())
	sqFilter2 := pipeApi.NewParallelFilter(NewSqWorker())

	pipeline := pipeApi.NewPipeline(genFilter, sqFilter1, sqFilter2)

	// This is a dummy generator (GenWorker overrides it)
	ch := make(chan interface{}, 100)
	for i := 0; i < 100; i++ {
		ch <- i
	}
	close (ch)

	for n := range pipeline.Filter(ch) {
		fmt.Print("|", n, "|")
	}
}

func DoConcurrentSimpleApi() {

	singleFilter := pipeApi.NewParallelFilter(NewGenWorker(100), NewSqWorker(), NewSqWorker())

	pipeline := pipeApi.NewPipeline(singleFilter)

	// This is a dummy generator (GenWorker overrides it)
	ch := make(chan interface{}, 100)
	for i := 0; i < 100; i++ {
		ch <- i
	}
	close (ch)

	for n := range pipeline.Filter(ch) {
		fmt.Print("|", n, "|")
	}
}
