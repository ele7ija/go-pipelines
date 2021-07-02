package internal

import (
	"fmt"
	"sync"
)

type Filter interface {
	Filter(in <-chan interface{}) <-chan interface{}
}

type Worker interface {
	Work(in interface{}) interface{}
}

// SerialFilter is filters a single item at a time
type SerialFilter struct {

	workers []Worker
}

func NewSerialFilter(workers ...Worker) *SerialFilter {
	return &SerialFilter{workers}
}

func (f*SerialFilter) Filter(in <-chan interface{}) <-chan interface{} {

	out := make(chan interface{})
	go func() {
		for nInterface := range in {
			out <- f.pipe(nInterface, 0)
		}
		fmt.Println("\n...Finishing SerialFilter")
		close(out)
	}()
	return out
}

func (f *SerialFilter) pipe(in interface{}, index int) interface{} {

	if index == len(f.workers) - 1 {
		return f.workers[index].Work(in)
	}
	return f.pipe(f.workers[index].Work(in), index + 1)
}

// ParallelFilter can filter multiple items at a time
type ParallelFilter struct {

	workers []Worker
}

func NewParallelFilter(workers ...Worker) *ParallelFilter {
	return &ParallelFilter{workers}
}

func (f*ParallelFilter) Filter(in <-chan interface{}) <-chan interface{} {

	out := make(chan interface{})
	wg := sync.WaitGroup{}
	go func() {
		for nInterface := range in {
			wg.Add(1)
			go func(nInterface interface{}) {
				out <- f.pipe(nInterface, 0)
				wg.Done()
			}(nInterface)
		}
		wg.Wait()
		close(out)
	}()
	return out
}

func (f *ParallelFilter) pipe(in interface{}, index int) interface{} {

	if index == len(f.workers) - 1 {
		return f.workers[index].Work(in)
	}
	return f.pipe(f.workers[index].Work(in), index + 1)
}

// Pipeline aggregates multiple Filter
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

func (p*Pipeline) AddFilter(filter Filter)  {

	p.filters = append(p.filters, filter)
}

func (p*Pipeline) Filter(in <-chan interface{}) <-chan interface{} {

	if len(p.filters) == 0 {
		emptych := make(chan interface{})
		close(emptych)
		return emptych
	}
	return p.pipe(in, 0)
}

func (p*Pipeline) pipe(in <-chan interface{}, index int) <-chan interface{} {

	if index == len(p.filters) - 1 {
		return p.filters[len(p.filters) - 1].Filter(in)
	}
	return p.pipe(p.filters[index].Filter(in), index + 1)
}


