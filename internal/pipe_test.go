package internal

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"
)

// SqWorker is a worker used for testing. It squares numbers.
type SqWorker struct {
}

func (f*SqWorker) Work(ctx context.Context, in Item) (Item, error) {

	time.Sleep(time.Millisecond * 2)
	n := in.(int)
	return n*n, nil
}

func TestNewSerialFilter(t *testing.T) {

	t.Run("default", func(t *testing.T) {

		no := 5
		var workers []Worker
		var name string
		for i := 0; i < no; i++ {
			worker := &SqWorker{}
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

	t.Run("default", func(t *testing.T) {
		f := NewSerialFilter(&SqWorker{})
		no := 100
		items := make(chan Item, no)
		errors := make(chan error, no)
		for i := 0; i < no; i++ {
			items <- i
		}
		close(items)
		filteredItems := f.Filter(context.Background(), items, errors)
		go func() {
			for err := range errors {
				t.Errorf("%s", err)
			}
		}()

		maxval := float64((no-1) * (no-1))
		var filteredItemsList []int
		for item := range filteredItems {
			num, ok := item.(int)
			if ok != true {
				t.Errorf("not a number at the end of the filter")
			}
			if math.Sqrt(float64(num)) > maxval {
				t.Errorf("Got a number larger than max")
			}
			if len(filteredItemsList) == 0 {
				if num != 0 {
					t.Errorf("first num should be 0")
				}
				filteredItemsList = append(filteredItemsList, num)
			} else {
				prevRoot := int(math.Sqrt(float64(num))) - 1
				if filteredItemsList[len(filteredItemsList) - 1] != prevRoot * prevRoot {
					t.Errorf("numbers aren't sorted")
				}
				filteredItemsList = append(filteredItemsList, num)
			}
		}
		close(errors)
	})
}

func TestBoundedParallelFilter_Filter(t *testing.T) {

	t.Run("sanity check", func(t *testing.T) {

		f := NewBoundedParallelFilter(5, &SqWorker{})
		no := 100
		items := make(chan Item, no)
		errors := make(chan error, no)
		for i := 0; i < no; i++ {
			items <- i
		}
		close(items)
		filteredItems := f.Filter(context.Background(), items, errors)
		go func() {
			for err := range errors {
				t.Errorf("%s", err)
			}
		}()

		maxval := float64((no-1) * (no-1))
		setOfFilteredItems := make(map[int]bool, no)
		for item := range filteredItems {
			num, ok := item.(int)
			if ok != true {
				t.Errorf("not a number at the end of the filter")
			}
			if setOfFilteredItems[num] {
				t.Errorf("already had this number")
			}
			if math.Sqrt(float64(num)) > maxval {
				t.Errorf("Got a number larger than max")
			}
			setOfFilteredItems[num] = true
		}
		close(errors)
	})
}


