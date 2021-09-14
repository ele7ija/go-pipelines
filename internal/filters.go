package internal

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// SerialFilter filters a single item at a time as they come
type SerialFilter struct {

	workers []Worker
	stat FilterExecutionStat
}

func NewSerialFilter(workers ...Worker) *SerialFilter {

	var filterName string
	for i, worker := range workers {
		if i == len(workers) - 1 {
			filterName += fmt.Sprintf("%T", worker)
		} else {
			filterName += fmt.Sprintf("%T,", worker)
		}
	}
	return &SerialFilter{workers, FilterExecutionStat{
		FilterName: filterName,
		FilterType: "SerialFilter"}}
}

func (f*SerialFilter) Filter(ctx context.Context, in <-chan Item, errors chan<- error) <-chan Item {

	items := make(chan Item)
	go func() {
		startedTotal := time.Now()
		for nInterface := range in {
			atomic.AddUint64(&f.stat.NumberOfItems, 1)
			started := time.Now()
			item, err := f.pipe(ctx, nInterface, 0)
			f.stat.TotalWork += time.Since(started)
			started = time.Now()
			if err != nil {
				errors <- err
			} else {
				items <- item
			}
			f.stat.TotalWaiting += time.Since(started)
		}
		f.stat.TotalDuration += time.Since(startedTotal)
		close(items)
	}()
	return items
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

func (f *SerialFilter) GetStat() FilterExecutionStat {

	return f.stat
}

// ParallelFilter can filter multiple items at a time
type ParallelFilter struct {

	workers []Worker
	stat FilterExecutionStat
}

func NewParallelFilter(workers ...Worker) *ParallelFilter {
	var filterName string
	for i, worker := range workers {
		if i == len(workers) - 1 {
			filterName += fmt.Sprintf("%T", worker)
		} else {
			filterName += fmt.Sprintf("%T,", worker)
		}
	}
	return &ParallelFilter{workers, FilterExecutionStat{
		FilterName:          filterName,
		FilterType: 		 "ParallelFilter",
	}}
}

func (f*ParallelFilter) Filter(ctx context.Context, in <-chan Item, errors chan<- error) <-chan Item {

	items := make(chan Item)
	wg := sync.WaitGroup{}
	go func() {
		startedTotal := time.Now()
		for item := range in {
			wg.Add(1)
			go func(item Item) {
				atomic.AddUint64(&f.stat.NumberOfItems, 1)
				started := time.Now()
				item, err := f.pipe(ctx, item, 0)
				f.stat.TotalWork += time.Since(started)
				started = time.Now()
				if err != nil {
					errors <- err
				} else {
					items <- item
				}
				f.stat.TotalWaiting += time.Since(started)
				wg.Done()
			}(item)
		}
		wg.Wait()
		f.stat.TotalDuration += time.Since(startedTotal)
		close(items)
	}()
	return items
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

func (f *ParallelFilter) GetStat() FilterExecutionStat {

	return f.stat
}

// BoundedParallelFilter can filter up to N items at a time
type BoundedParallelFilter struct {

	sem chan struct{} // Semaphore implementation
	workers []Worker
	stat FilterExecutionStat
}

func NewBoundedParallelFilter(bound int, workers ...Worker) *BoundedParallelFilter {

	var filterName string
	for i, worker := range workers {
		if i == len(workers) - 1 {
			filterName += fmt.Sprintf("%T", worker)
		} else {
			filterName += fmt.Sprintf("%T,", worker)
		}
	}
	return &BoundedParallelFilter{
		make(chan struct{}, bound),
		workers,
		FilterExecutionStat{
			FilterName:          filterName,
			FilterType: 		 "BoundedParallelFilter"}}
}

func (f*BoundedParallelFilter) Filter(ctx context.Context, in <-chan Item, errors chan<- error) <-chan Item {

	items := make(chan Item)
	wg := sync.WaitGroup{}
	go func() {
		startedTotal := time.Now()
		for item := range in {
			f.sem <- struct{}{}
			wg.Add(1)
			go func(item Item) {
				atomic.AddUint64(&f.stat.NumberOfItems, 1)
				started := time.Now()
				item, err := f.pipe(ctx, item, 0)
				f.stat.TotalWork += time.Since(started)
				started = time.Now()
				if err != nil {
					errors <- err
				} else {
					items <- item
				}
				f.stat.TotalWaiting += time.Since(started)
				<-f.sem
				wg.Done()
			}(item)
		}
		wg.Wait()
		f.stat.TotalDuration += time.Since(startedTotal)
		close(items)
	}()
	return items
}

func (f *BoundedParallelFilter) pipe(ctx context.Context, in Item, index int) (Item, error) {

	out, err := f.workers[index].Work(ctx, in)
	if err != nil {
		return nil, err
	}

	if index == len(f.workers) - 1 {
		return out, nil
	}

	return f.pipe(ctx, out, index + 1)
}

func (f *BoundedParallelFilter) GetStat() FilterExecutionStat {

	return f.stat
}

// IndependentSerialFilter filters one item at a time and sends it only when it filtered all of them.
type IndependentSerialFilter struct {

	workers []Worker
	stat FilterExecutionStat
}

func NewIndependentSerialFilter(workers ...Worker) *IndependentSerialFilter {

	var filterName string
	for i, worker := range workers {
		if i == len(workers) - 1 {
			filterName += fmt.Sprintf("%T", worker)
		} else {
			filterName += fmt.Sprintf("%T,", worker)
		}
	}
	return &IndependentSerialFilter{workers, FilterExecutionStat{
		FilterName: filterName,
		FilterType: "IndependentSerialFilter"}}
}

func (f*IndependentSerialFilter) Filter(ctx context.Context, in <-chan Item, errors chan<- error) <-chan Item {

	items := make(chan Item)
	var itemsBuffer []Item
	go func() {
		startedTotal := time.Now()
		for nInterface := range in {
			atomic.AddUint64(&f.stat.NumberOfItems, 1)
			started := time.Now()
			item, err := f.pipe(ctx, nInterface, 0)
			f.stat.TotalWork += time.Since(started)
			started = time.Now()
			if err != nil {
				errors <- err
			} else {
				// don't write it right away
				itemsBuffer = append(itemsBuffer, item)
			}
			f.stat.TotalWaiting += time.Since(started)
		}
		for _, item := range itemsBuffer {
			items <- item
		}
		close(items)
		f.stat.TotalDuration += time.Since(startedTotal)
	}()
	return items
}

func (f *IndependentSerialFilter) pipe(ctx context.Context, in Item, index int) (Item, error) {

	out, err := f.workers[index].Work(ctx, in)
	if err != nil {
		return nil, err
	}

	if index == len(f.workers) - 1 {
		return out, nil
	}

	return f.pipe(ctx, out, index + 1)
}

func (f *IndependentSerialFilter) GetStat() FilterExecutionStat {

	return f.stat
}
