package examples

import (
	"context"
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

func (f*GenWorker) Work(ctx context.Context, in pipeApi.Item) (pipeApi.Item, error) {

	time.Sleep(time.Millisecond * 2)
	curr := f.curr
	f.curr++
	item := curr
	return item, nil
}

type SqWorker struct {
}

func NewSqWorker() *SqWorker {

	return &SqWorker{}
}

func (f*SqWorker) Work(ctx context.Context, in pipeApi.Item) (pipeApi.Item, error) {

	time.Sleep(time.Millisecond * 2)
	n := in.(int)
	return n*n, nil
}

func DoConcurrentApi() {

	genFilter := pipeApi.NewParallelFilter(NewGenWorker(100))
	sqFilter1 := pipeApi.NewParallelFilter(NewSqWorker())
	sqFilter2 := pipeApi.NewParallelFilter(NewSqWorker())

	pipeline := pipeApi.NewPipeline("Etw", genFilter, sqFilter1, sqFilter2)

	// This is a dummy generator (GenWorker overrides it)
	ch := make(chan pipeApi.Item, 100)
	for i := 0; i < 100; i++ {
		ch <- i
	}
	close (ch)
	errors := make(chan error, 100)
	items := pipeline.Filter(context.Background(), ch, errors)
	go func() {
		for err := range errors {
			fmt.Printf("Unexpected error: %s", err)
		}
	}()
	for n := range items {
		fmt.Print("|", n, "|")
	}
	close(errors)

}

func DoConcurrentSimpleApi() {

	singleFilter := pipeApi.NewParallelFilter(NewGenWorker(100), NewSqWorker(), NewSqWorker())

	pipeline := pipeApi.NewPipeline("Etw", singleFilter)

	// This is a dummy generator (GenWorker overrides it)
	ch := make(chan pipeApi.Item, 100)
	for i := 0; i < 100; i++ {
		ch <- i
	}
	close (ch)
	errors := make(chan error, 100)
	items := pipeline.Filter(context.Background(), ch, errors)
	go func() {
		for err := range errors {
			fmt.Printf("Unexpected error: %s", err)
		}
	}()
	for n := range items {
		fmt.Print("|", n.(int), "|")
	}
	close(errors)
}
