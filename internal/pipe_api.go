package internal

import (
	"context"
	"log"
	"sync"
)

type Item interface {
	AddPart(part interface{})
	GetPart(index int) interface{}
	NumberOfParts() int
}

type Worker interface {
	Work(ctx context.Context, in Item) (Item, error)
}

type Filter interface {
	Filter(ctx context.Context, in <-chan Item) (<-chan Item, <-chan error)
}

// SerialFilter filters a single item at a time
type SerialFilter struct {

	workers []Worker
}

type ItemImpl []interface{}

func NewGenericItem(parts ...interface{}) Item {
	item := ItemImpl{}
	for _, part := range parts {
		item.AddPart(part)
	}
	return &item
}

func (i *ItemImpl) AddPart(newInput interface{}) {
	*i = append(*i, newInput)
}

func (i *ItemImpl) GetPart(index int) interface{} {
	return (*i)[index]
}

func (i ItemImpl) NumberOfParts() int {
	return len(i)
}

func NewSerialFilter(workers ...Worker) *SerialFilter {
	return &SerialFilter{workers}
}

func (f*SerialFilter) Filter(ctx context.Context, in <-chan Item) (<-chan Item, <-chan error) {

	items := make(chan Item)
	errors := make(chan error)
	go func() {
		for nInterface := range in {
			item, err := f.pipe(ctx, nInterface, 0)
			if err != nil {
				errors <- err
			} else {
				items <- item
			}
		}
		close(items)
		close(errors)
	}()
	return items, errors
}

func (f *SerialFilter) pipe(ctx context.Context, in Item, index int) (Item, error) {

	out, err := f.workers[index].Work(ctx, in)
	if err != nil {
		return nil, err
	}

	if index == len(f.workers) - 1 {
		return out, nil
	}

	return f.pipe(ctx, out, index + 1)
}

// ParallelFilter can filter multiple items at a time
type ParallelFilter struct {

	workers []Worker
}

func NewParallelFilter(workers ...Worker) *ParallelFilter {
	return &ParallelFilter{workers}
}

func (f*ParallelFilter) Filter(ctx context.Context, in <-chan Item) (<-chan Item, <-chan error) {

	items := make(chan Item)
	errors := make(chan error)
	wg := sync.WaitGroup{}
	go func() {
		for item := range in {
			wg.Add(1)
			go func(item Item) {
				item, err := f.pipe(ctx, item, 0)
				if err != nil {
					errors <- err
				} else {
					items <- item
				}
				wg.Done()
			}(item)
		}
		wg.Wait()
		close(items)
		close(errors)
	}()
	return items, errors
}

func (f *ParallelFilter) pipe(ctx context.Context, in Item, index int) (Item, error) {

	out, err := f.workers[index].Work(ctx, in)
	if err != nil {
		return nil, err
	}

	if index == len(f.workers) - 1 {
		return out, nil
	}

	return f.pipe(ctx, out, index + 1)
}

// Pipeline aggregates multiple Filter
type Pipeline struct {

	filters []Filter
	errors chan error
}

func NewPipeline(filters ...Filter) *Pipeline {

	pipeline := Pipeline{}
	for _, filter := range filters {
		pipeline.AddFilter(filter)
	}
	pipeline.errors = make(chan error)
	return &pipeline
}

func (p*Pipeline) AddFilter(filter Filter)  {

	p.filters = append(p.filters, filter)
}

func (p*Pipeline) Filter(ctx context.Context, in <-chan Item) (<-chan Item, <-chan error) {

	log.Printf("Starting the pipeline...")
	if len(p.filters) == 0 {
		emptych := make(chan Item)
		emptycherr := make(chan error)
		close(emptych)
		close(emptycherr)
		return emptych, emptycherr
	}
	return p.pipe(ctx, in, 0), p.errors
}

func (p*Pipeline) pipe(ctx context.Context, in <-chan Item, index int) <-chan Item {

	log.Printf("Going through the filter number: %d", index)
	items, errors := p.filters[index].Filter(ctx, in)
	go func() {
		for err := range errors {
			p.errors <- err
		}
		if index == len(p.filters) - 1 {
			close(p.errors)
		}
	}()

	if index == len(p.filters) - 1 {
		return items
	}

	return p.pipe(ctx, items, index + 1)
}


