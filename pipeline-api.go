package go_pipelines

import (
	"fmt"
	"sync"
	"time"
)

type Filter interface {
	Filter(in <-chan interface{}) <-chan interface{}
}

type Worker interface {
	Work(in interface{}) interface{}
}

type SerialFilter struct {
	worker Worker
}

func NewSerialFilter(worker Worker) *SerialFilter {
	return &SerialFilter{worker}
}

func (f* SerialFilter) Filter(in <-chan interface{}) <-chan interface{} {

	fmt.Println("\nStarting SerialFilter...")
	out := make(chan interface{})
	go func() {
		for nInterface := range in {
			out <- f.worker.Work(nInterface)
		}
		fmt.Println("\n...Finishing SerialFilter")
		close(out)
	}()
	return out
}

type ParallelFilter struct {

	worker Worker
}

func NewParallelFilter(worker Worker) *ParallelFilter {
	return &ParallelFilter{worker}
}

func (f* ParallelFilter) Filter(in <-chan interface{}) <-chan interface{} {

	fmt.Println("\nStarting ParallelFilter...")
	out := make(chan interface{})
	wg := sync.WaitGroup{}
	go func() {
		for nInterface := range in {
			wg.Add(1)
			go func(nInterface interface{}) {
				out <- f.worker.Work(nInterface)
				wg.Done()
			}(nInterface)
		}
		wg.Wait()
		fmt.Println("\nFinishing ParallelFilter")
		close(out)
	}()
	return out
}

type Pipeline struct {

	filters []Filter
}

func NewPipeline(filters ...Filter) *Pipeline {

	pipeline := Pipeline{}
	for _, filter := range filters {
		pipeline.AddFilter(filter)
	}
	return &pipeline
}

func (p* Pipeline) AddFilter(filter Filter)  {

	p.filters = append(p.filters, filter)
}

func (p* Pipeline) Filter(in <-chan interface{}) <-chan interface{} {

	if len(p.filters) == 0 {
		emptych := make(chan interface{})
		close(emptych)
		return emptych
	}
	return p.pipe(in, 0)
}

func (p* Pipeline) pipe(in <-chan interface{}, index int) <-chan interface{} {

	fmt.Println("Piping filter: ", index)
	if index == len(p.filters) - 1 {
		return p.filters[len(p.filters) - 1].Filter(in)
	}
	return p.pipe(p.filters[index].Filter(in), index + 1)
}

type GenWorker struct {

	max int
	curr int
}

func NewGenWorker(max int) *GenWorker {
	return &GenWorker{max: max, curr: 0}
}

func (f* GenWorker) Work(in interface{}) interface{} {

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

func (f* SqWorker) Work(in interface{}) interface{} {

	time.Sleep(time.Millisecond * 2)
	n := in.(int)
	return n * n
}


