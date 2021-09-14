package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	NoError = -1
	EveryError = 0
	EveryOtherError = 1
)

// I'm just here for testing. I bit-shift to left.
type LBitShiftWorker struct {
	errorInterval int64
	counter int64
	mu sync.Mutex
}

func NewLBitShiftWorker(errorInterval int64) *LBitShiftWorker {
	return &LBitShiftWorker{errorInterval: errorInterval}
}

func (f*LBitShiftWorker) Work(ctx context.Context, in Item) (Item, error) {

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.counter == f.errorInterval {
		f.counter = 0
		return nil, fmt.Errorf("artificial error")
	}
	f.counter++
	n := in.(int)
	return n << 1, nil
}

// I'm just here for testing. I shift bits to left N times, one item at a time.
type LBitShiftFilter struct {
	workers []Worker
	stat FilterExecutionStat
}

func NewLBitShiftFilter(no int) *LBitShiftFilter {

	var workers []Worker
	for i := 0; i < no; i++ {
		workers = append(workers, NewLBitShiftWorker(-1))
	}
	return &LBitShiftFilter{workers, FilterExecutionStat{
		FilterName: fmt.Sprintf("%d workers", no),
		FilterType: "LBitShiftFilter"}}
}

func (f*LBitShiftFilter) Filter(ctx context.Context, in <-chan Item, errors chan<- error) <-chan Item {

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

func (f *LBitShiftFilter) pipe(ctx context.Context, in Item, index int) (Item, error) {

	out, err := f.workers[index].Work(ctx, in)
	if err != nil {
		return nil, err
	}

	if index == len(f.workers) - 1 {
		return out, nil
	}

	return f.pipe(ctx, out, index + 1)
}

func (f *LBitShiftFilter) GetStat() FilterExecutionStat {

	return f.stat
}

func TestNewPipeline(t *testing.T) {

	pipelineTestName := "testPipeline"
	filterNos := []int{0, 1, 5, 10}

	for _, noFilters := range filterNos {
		t.Run(fmt.Sprintf("%d filters", noFilters), func(t *testing.T) {
			var filters []Filter
			for i := 0; i < noFilters; i++ {
				filters = append(filters, NewLBitShiftFilter(0))
			}
			p := NewPipeline(pipelineTestName, filters...)

			if len(p.filters) != noFilters {
				t.Errorf("incorrect number of filters")
			}
			if p.name != pipelineTestName {
				t.Errorf("incorrect name")
			}
		})
	}
}

func TestPipeline_Filter(t *testing.T) {

	pipelineTestName := "testPipeline"
	filterNos := []int{0, 1, 5, 10}
	itemNos := []int{0, 1, 10}

	for _, noFilters := range filterNos {
		for _, noItems := range itemNos {
			t.Run(fmt.Sprintf("%d filters, %d items", noFilters, noItems), func(t *testing.T) {
				var filters []Filter
				for i := 0; i < noFilters; i++ {
					filters = append(filters, NewLBitShiftFilter(1))
				}
				p := NewPipeline(pipelineTestName, filters...)

				items := make(chan Item, noItems)
				errors := make(chan error, noItems)
				for i := 0; i < noItems; i++ {
					items <- i
				}
				close(items)
				filteredItems := p.Filter(context.Background(), items, errors)
				go func() {
					for err := range errors {
						t.Errorf("encountered an unexpected error: %s", err)
					}
				}()
				setOfFilteredItems := make(map[int]bool, noItems)
				maxval := (noItems - 1) << noFilters
				for filteredItem := range filteredItems {
					item := filteredItem.(int)
					if setOfFilteredItems[item] {
						t.Errorf("This item was already here")
					}
					if item > maxval {
						t.Errorf("Item larger than maxval")
					}
					setOfFilteredItems[item] = true
				}
				close(errors)
			})
		}
	}
}

func TestPipeline_SaveStats(t *testing.T) {

	pipelineTestName := "testPipeline"
	noFilters := 1

	t.Run("default", func(t *testing.T) {
		var filters []Filter
		for i := 0; i < noFilters; i++ {
			filters = append(filters, NewLBitShiftFilter(1))
		}
		p := NewPipeline(pipelineTestName, filters...)
		err := p.SaveStats()
		if err != nil {
			t.Errorf("error: %s", err)
		}
		statsFile, err := os.Open(p.statPath)
		if err != nil {
			t.Errorf("err: %s", err)
		}
		var stat PipelineStat
		err = json.NewDecoder(statsFile).Decode(&stat)
		if err != nil {
			t.Errorf("Wrongly written stats: %s", err)
		}
		if stat.PipelineName != p.name ||
			len(stat.FilterStats) != len(p.filters) {
			t.Errorf("Wrong stats: %s", err)
		}
		err = os.Remove(p.statPath)
		if err != nil {
			t.Logf("Couldn't erase temp file: %s, err: %s", p.statPath, err)
		}
	})
}

func TestPipeline_StartExtracting(t *testing.T) {

	pipelineTestName := "testPipeline"
	noFilters := 1

	t.Run("default", func(t *testing.T) {
		dur := time.Second

		var filters []Filter
		for i := 0; i < noFilters; i++ {
			filters = append(filters, NewLBitShiftFilter(1))
		}
		p := NewPipeline(pipelineTestName, filters...)
		p.StartExtracting(dur)
		time.Sleep(dur + time.Millisecond * 100)
		if p.statPath == "" {
			t.Errorf("Stat path should've been changed")
		}
		statsFile, err := os.Open(p.statPath)
		if err != nil {
			t.Errorf("err: %s", err)
		}
		var stat PipelineStat
		err = json.NewDecoder(statsFile).Decode(&stat)
		if err != nil {
			t.Errorf("Wrongly written stats: %s", err)
		}
		if stat.PipelineName != p.name ||
			len(stat.FilterStats) != len(p.filters) {
			t.Errorf("Wrong stats: %s", err)
		}
		err = os.Remove(p.statPath)
		if err != nil {
			t.Logf("Couldn't erase temp file: %s, err: %s", p.statPath, err)
		}
	})
}