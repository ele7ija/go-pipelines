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

	genFilter := pipeApi.NewSerialFilter(NewGenWorker(100))
	sqFilter1 := pipeApi.NewParallelFilter(NewSqWorker())
	sqFilter2 := pipeApi.NewParallelFilter(NewSqWorker())

	pipeline := pipeApi.NewPipeline(genFilter, sqFilter1, sqFilter2)

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
