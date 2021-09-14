package internal

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func TestNewSerialFilter(t *testing.T) {

	t.Run("default", func(t *testing.T) {

		no := 5
		var workers []Worker
		var name string
		for i := 0; i < no; i++ {
			worker := NewLBitShiftWorker(-1)
			workers = append(workers, worker)
			if i == no - 1 {
				name += fmt.Sprintf("%T", worker)
			} else {
				name += fmt.Sprintf("%T,", worker)
			}
		}
		sf := NewSerialFilter(workers...)

		if len(sf.workers) != no {
			t.Errorf("err")
		}
		if sf.GetStat().FilterName != name {
			t.Errorf("err")
		}
	})
}

func TestSerialFilter_Filter(t *testing.T) {

	errorIntervals := []int64{NoError, EveryError, EveryOtherError, 50}
	for _, errorInterval := range errorIntervals {
		t.Run(fmt.Sprintf("error interval of %d", errorInterval), func(t *testing.T) {
			f := NewSerialFilter(NewLBitShiftWorker(errorInterval))
			no := 100
			items := make(chan Item, no)
			errors := make(chan error, no)
			for i := 0; i < no; i++ {
				items <- i
			}
			close(items)
			filteredItems := f.Filter(context.Background(), items, errors)
			var errorCounter int64
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				for range errors {
					atomic.AddInt64(&errorCounter, 1)
				}
				wg.Done()
			}()

			maxval := float64(100 << 1)
			var filteredItemsList []int
			for item := range filteredItems {
				num, ok := item.(int)
				if ok != true {
					t.Errorf("not a number at the end of the filter")
				}
				if float64(num) > maxval {
					t.Errorf("Got a number larger than max")
				}
				if len(filteredItemsList) == 0 {
					if num != 0 {
						t.Errorf("first num should be 0")
					}
					filteredItemsList = append(filteredItemsList, num)
				} else {
					if filteredItemsList[len(filteredItemsList) - 1] >= num {
						t.Errorf("numbers aren't sorted")
					}
					filteredItemsList = append(filteredItemsList, num)
				}
			}
			close(errors)
			wg.Wait()
			if errorInterval == NoError && errorCounter != 0 {
				t.Errorf("bad number of errors")
			}
			if errorInterval == EveryError && errorCounter != int64(no) {
				t.Errorf("bad number of errors")
			}
			if errorInterval != NoError && errorInterval != EveryError {
				shouldBeErrors := int64(no) / (errorInterval + 1)
				if shouldBeErrors != errorCounter {
					t.Errorf("bad number of errors")
				}
			}

		})
	}

}

func TestParallelFilter_Filter(t *testing.T) {

	errorIntervals := []int64{NoError, EveryError, EveryOtherError, 50}
	for _, errorInterval := range errorIntervals {
		t.Run(fmt.Sprintf("error interval of %d", errorInterval), func(t *testing.T) {
			f := NewParallelFilter(NewLBitShiftWorker(errorInterval))
			no := 100
			items := make(chan Item, no)
			errors := make(chan error, no)
			for i := 0; i < no; i++ {
				items <- i
			}
			close(items)
			filteredItems := f.Filter(context.Background(), items, errors)
			var errorCounter int64
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				for range errors {
					atomic.AddInt64(&errorCounter, 1)
				}
				wg.Done()
			}()

			var itemCounter int
			maxval := float64(100 << 1)
			setOfFilteredItems := make(map[int]bool, no)
			for item := range filteredItems {
				itemCounter++
				num, ok := item.(int)
				if ok != true {
					t.Errorf("not a number at the end of the filter")
				}
				if float64(num) > maxval {
					t.Errorf("Got a number larger than max")
				}
				if setOfFilteredItems[num] {
					t.Errorf("already had this number")
				}
				if float64(num) > maxval {
					t.Errorf("Got a number larger than max")
				}
				setOfFilteredItems[num] = true
			}
			close(errors)
			wg.Wait()
			if errorInterval == NoError && errorCounter != 0 {
				t.Errorf("bad number of errors")
			}
			if errorInterval == EveryError && errorCounter != int64(no) {
				t.Errorf("bad number of errors")
			}
			if errorInterval != NoError && errorInterval != EveryError {
				shouldBeErrors := int64(no) / (errorInterval + 1)
				if shouldBeErrors != int64(errorCounter) {
					t.Errorf("bad number of errors: should be %d vs got %d, items no: %d", shouldBeErrors, int64(errorCounter), itemCounter)
				}
			}

		})
	}

}

func TestBoundedParallelFilter_Filter(t *testing.T) {
	errorIntervals := []int64{NoError, EveryError, EveryOtherError, 50}
	for _, errorInterval := range errorIntervals {
		t.Run(fmt.Sprintf("error interval of %d", errorInterval), func(t *testing.T) {
			f := NewBoundedParallelFilter(1, NewLBitShiftWorker(errorInterval))
			no := 100
			items := make(chan Item, no)
			errors := make(chan error, no)
			for i := 0; i < no; i++ {
				items <- i
			}
			close(items)
			filteredItems := f.Filter(context.Background(), items, errors)
			var errorCounter int
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				for range errors {
					errorCounter++
				}
				wg.Done()
			}()

			maxval := float64(100 << 1)
			setOfFilteredItems := make(map[int]bool, no)
			for item := range filteredItems {
				num, ok := item.(int)
				if ok != true {
					t.Errorf("not a number at the end of the filter")
				}
				if float64(num) > maxval {
					t.Errorf("Got a number larger than max")
				}
				if setOfFilteredItems[num] {
					t.Errorf("already had this number")
				}
				if float64(num) > maxval {
					t.Errorf("Got a number larger than max")
				}
				setOfFilteredItems[num] = true
			}
			close(errors)
			wg.Wait()
			if errorInterval == NoError && errorCounter != 0 {
				t.Errorf("bad number of errors")
			}
			if errorInterval == EveryError && errorCounter != no {
				t.Errorf("bad number of errors")
			}
			if errorInterval != NoError && errorInterval != EveryError {
				shouldBeErrors := int64(no) / (errorInterval + 1)
				if shouldBeErrors != int64(errorCounter) {
					t.Errorf("bad number of errors")
				}
			}

		})
	}

}

func TestIndependentSerialFilter_Filter(t *testing.T) {

	errorIntervals := []int64{NoError, EveryError, EveryOtherError, 50}
	for _, errorInterval := range errorIntervals {
		t.Run(fmt.Sprintf("error interval of %d", errorInterval), func(t *testing.T) {
			f := NewIndependentSerialFilter(NewLBitShiftWorker(errorInterval))
			no := 100
			items := make(chan Item, no)
			errors := make(chan error, no)
			for i := 0; i < no; i++ {
				items <- i
			}
			close(items)
			filteredItems := f.Filter(context.Background(), items, errors)
			var errorCounter int
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				for range errors {
					errorCounter++
				}
				wg.Done()
			}()

			maxval := float64(100 << 1)
			setOfFilteredItems := make(map[int]bool, no)
			for item := range filteredItems {
				num, ok := item.(int)
				if ok != true {
					t.Errorf("not a number at the end of the filter")
				}
				if float64(num) > maxval {
					t.Errorf("Got a number larger than max")
				}
				if setOfFilteredItems[num] {
					t.Errorf("already had this number")
				}
				if float64(num) > maxval {
					t.Errorf("Got a number larger than max")
				}
				setOfFilteredItems[num] = true
			}
			close(errors)
			wg.Wait()
			if errorInterval == NoError && errorCounter != 0 {
				t.Errorf("bad number of errors")
			}
			if errorInterval == EveryError && errorCounter != no {
				t.Errorf("bad number of errors")
			}
			if errorInterval != NoError && errorInterval != EveryError {
				shouldBeErrors := int64(no) / (errorInterval + 1)
				if shouldBeErrors != int64(errorCounter) {
					t.Errorf("bad number of errors")
				}
			}

		})
	}
}


