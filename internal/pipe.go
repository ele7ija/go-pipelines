package internal

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type Item interface{}

type Worker interface {
	Work(ctx context.Context, in Item) (Item, error)
}

type Filter interface {
	Filter(ctx context.Context, in <-chan Item, errors chan<- error) <-chan Item // error channel is a param so as not to keep errors internally
	GetStat() FilterExecutionStat
}

// SerialFilter filters a single item at a time
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

// Pipeline aggregates multiple Filter
type Pipeline struct {

	name string
	filters []Filter
	statPath string

	FilteringDuration time.Duration
	FilteringNumber int
}

func NewPipeline(name string, filters ...Filter) *Pipeline {

	pipeline := Pipeline{name: name}
	for _, filter := range filters {
		pipeline.AddFilter(filter)
	}
	return &pipeline
}

func (p*Pipeline) AddFilter(filter Filter)  {

	p.filters = append(p.filters, filter)
}

func (p*Pipeline) Filter(ctx context.Context, in <-chan Item, errors chan<- error) <-chan Item {

	if len(p.filters) == 0 {
		emptych := make(chan Item)
		close(emptych)
		close(errors)
		return emptych
	}
	items := p.pipe(ctx, in, 0, errors)
	return items
}

func (p*Pipeline) pipe(ctx context.Context, in <-chan Item, index int, errors chan<- error) <-chan Item {

	items := p.filters[index].Filter(ctx, in, errors)
	if index == len(p.filters) - 1 {
		return items
	}

	return p.pipe(ctx, items, index + 1, errors)
}

type FilterExecutionStat struct {

	FilterName string
	FilterType string
	TotalDuration time.Duration
	TotalWork time.Duration
	TotalWaiting time.Duration
	NumberOfItems uint64
}

type PipelineStat struct {

	PipelineName		   string
	FilterStats            []FilterExecutionStat
	TotalDuration          time.Duration
	TotalNumberOfFiltering int
}

func (p *Pipeline) SaveStats() []FilterExecutionStat  {

	stat := PipelineStat{
		TotalDuration:          p.FilteringDuration,
		TotalNumberOfFiltering: p.FilteringNumber,
		PipelineName:           p.name,
	}
	var stats []FilterExecutionStat
	for _, filter := range p.filters {
		stats = append(stats, filter.GetStat())
	}
	stat.FilterStats = stats

	var statsFile *os.File
	var err error
	if p.statPath == "" {
		statsFile, err = ioutil.TempFile(os.TempDir(), fmt.Sprintf("%s*.json", p.name))
		p.statPath = statsFile.Name()
		log.Debugf("Created a new file for stats: %s", p.statPath)
	} else {
		statsFile, err = os.OpenFile(p.statPath, os.O_WRONLY, os.ModeAppend)
	}
	if err != nil {
		log.Debugf("Error: %s", err)
	}
	err = json.NewEncoder(statsFile).Encode(stat)
	if err != nil {
		log.Debugf("Error: %s", err)
	}

	return stats
}

func (p *Pipeline) StartExtracting() {

	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			for range ticker.C {
				log.Debugf("Saving stats...")
				p.SaveStats()
			}
		}
	}()
}

func (p *Pipeline) GetStat() FilterExecutionStat {

	stat := FilterExecutionStat{
		FilterName:    p.name,
	}
	for i, filter := range p.filters {
		statTemp := filter.GetStat()
		if i == 0 {
			stat.NumberOfItems = statTemp.NumberOfItems
		}
		stat.TotalDuration += statTemp.TotalDuration
		stat.TotalWork += statTemp.TotalWork
		stat.TotalWaiting += statTemp.TotalWaiting
	}

	return stat
}


