package storage

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var flagvar int

func init() {
	flag.IntVar(&flagvar, "numts", 1000, "number of timeseries")
}

func TestWriteNormalThroughPut(t *testing.T) {
	scrapeCountBatch := 43200 // seconds, 12 hours
	num_ts := flagvar
	path := "BenchmarkStorageWriteThoughput"
	s := MustOpenStorage(path, 0, 0, 0)
	defer func() {
		s.MustClose()
		if err := os.RemoveAll(path); err != nil {
			t.Fatalf("cannot remove storage at %q: %s", path, err)
		}
	}()

	var mn MetricName
	metricRows := make([]MetricRow, num_ts)
	mn.MetricGroup = []byte("fake_metric")
	for ts_id := 0; ts_id < num_ts; ts_id++ {
		mn.Tags = []Tag{
			{[]byte("machine"), []byte(strconv.Itoa(ts_id))},
		}
		mr := &metricRows[ts_id]
		mr.MetricNameRaw = mn.marshalRaw(mr.MetricNameRaw[:0])
	}

	tNow := time.Now()
	ingestNormalScrapes(s, metricRows, scrapeCountBatch)
	since := time.Since(tNow)

	throughput := 43200.0 * float64(num_ts) / float64(since.Seconds())
	t.Log(num_ts, since.Seconds(), throughput)
}

func ingestNormalScrapes(st *Storage, mrs []MetricRow, scrapeTotCount int) {
	var wg sync.WaitGroup
	scrapeBatch := 100
	const second = 100
	var count atomic.Int64
	count.Store(0)

	for i := 0; i < scrapeTotCount; i += scrapeBatch {
		currTime := int64(i * second)
		lbls := mrs
		for len(lbls) > 0 {
			b := 1000
			batch := lbls[:b]
			lbls = lbls[b:]
			wg.Add(1)
			go func(currTime int64) {
				defer wg.Done()
				for j := 0; j < scrapeBatch; j++ {
					rowsToInsert := make([]MetricRow, 0, len(batch))
					ts := int64(j*second) + currTime
					for _, mr := range batch {
						mr.Value = rand.NormFloat64() * 100000
						mr.Timestamp = ts
						rowsToInsert = append(rowsToInsert, mr)
					}

					if err := st.AddRows(rowsToInsert, defaultPrecisionBits); err != nil {
						panic(fmt.Errorf("cannot add rows to storage: %w", err))
					}
				}
			}(currTime)
		}
	}

	wg.Wait()
	fmt.Println("ingestion completed")
}

func TestWriteZipfThroughPut(t *testing.T) {
	scrapeCountBatch := 43200 // seconds, 12 hours
	num_ts := flagvar
	path := "BenchmarkStorageWriteThoughput"
	s := MustOpenStorage(path, 0, 0, 0)
	defer func() {
		s.MustClose()
		if err := os.RemoveAll(path); err != nil {
			t.Fatalf("cannot remove storage at %q: %s", path, err)
		}
	}()

	var mn MetricName
	metricRows := make([]MetricRow, num_ts)
	mn.MetricGroup = []byte("fake_metric")
	for ts_id := 0; ts_id < num_ts; ts_id++ {
		mn.Tags = []Tag{
			{[]byte("machine"), []byte(strconv.Itoa(ts_id))},
		}
		mr := &metricRows[ts_id]
		mr.MetricNameRaw = mn.marshalRaw(mr.MetricNameRaw[:0])
	}

	tNow := time.Now()
	ingestZipfScrapes(s, metricRows, scrapeCountBatch)
	since := time.Since(tNow)

	throughput := 43200.0 * float64(num_ts) / float64(since.Seconds())
	t.Log(num_ts, since.Seconds(), throughput)
}

func ingestZipfScrapes(st *Storage, mrs []MetricRow, scrapeTotCount int) {
	var wg sync.WaitGroup
	scrapeBatch := 100
	const second = 100
	var count atomic.Int64
	count.Store(0)
	for i := 0; i < scrapeTotCount; i += scrapeBatch {
		currTime := int64(i * second)
		lbls := mrs
		for len(lbls) > 0 {
			b := 1000
			batch := lbls[:b]
			lbls = lbls[b:]
			wg.Add(1)
			go func(currTime int64) {
				defer wg.Done()

				var s float64 = 1.01
				var v float64 = 1
				var RAND *rand.Rand = rand.New(rand.NewSource(time.Now().Unix()))
				z := rand.NewZipf(RAND, s, v, uint64(100000))

				for j := 0; j < scrapeBatch; j++ {
					rowsToInsert := make([]MetricRow, 0, len(batch))
					ts := int64(j*second) + currTime
					for _, mr := range batch {
						mr.Value = float64(z.Uint64())
						mr.Timestamp = ts
						rowsToInsert = append(rowsToInsert, mr)
					}

					if err := st.AddRows(rowsToInsert, defaultPrecisionBits); err != nil {
						panic(fmt.Errorf("cannot add rows to storage: %w", err))
					}
				}
			}(currTime)
		}
	}

	wg.Wait()
	fmt.Println("ingestion completed")
}
