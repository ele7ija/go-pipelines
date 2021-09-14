package internal

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"time"
)

type Item interface{} // I'm processed in the pipeline.

type Worker interface {
	Work(ctx context.Context, in Item) (Item, error)
} // I process a single Item.

type Filter interface {
	Filter(ctx context.Context, in <-chan Item, errors chan<- error) <-chan Item
	GetStat() FilterExecutionStat
} // I process a bunch of items. I do not stop processing if there's an error.

type Pipeline struct {

	name string
	filters []Filter
	statPath string

	FilteringDuration time.Duration
	FilteringNumber int
} // I process a bunch of items. I contain filters, but I'm also a Filter!

func NewPipeline(name string, filters ...Filter) *Pipeline {

	pipeline := Pipeline{name: name}
	for _, filter := range filters {
		pipeline.filters = append(pipeline.filters, filter)
	}
	return &pipeline
}

func (p*Pipeline) Filter(ctx context.Context, in <-chan Item, errors chan<- error) <-chan Item {

	if len(p.filters) == 0 {
		emptych := make(chan Item)
		close(emptych)
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
} // I keep track of the performance of a Filter.

type PipelineStat struct {

	PipelineName		   string
	FilterStats            []FilterExecutionStat
	TotalDuration          time.Duration
	TotalNumberOfFiltering int
}

// SaveStats saves stats from filters to a temp file
func (p *Pipeline) SaveStats() error {

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
		if err == nil {
			p.statPath = statsFile.Name()
			log.Debugf("Created a new file for stats: %s", p.statPath)
		}
	} else {
		statsFile, err = os.OpenFile(p.statPath, os.O_WRONLY, os.ModeAppend)
	}
	if err != nil {
		return err
	}
	return json.NewEncoder(statsFile).Encode(stat)
}

func (p *Pipeline) StartExtracting(dur time.Duration) {

	ticker := time.NewTicker(dur)
	go func() {
		for range ticker.C {
			log.Debugf("Saving stats...")
			err := p.SaveStats()
			if err != nil {
				log.Debugf("Couldn't save stats: %s", err)
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


