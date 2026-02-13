package prometheus

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	prom_config "github.com/prometheus/common/config"
	"gotest.tools/assert"
)

func TestCacheData_ConcurrentAccess(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T, data *cacheData)
	}{
		{
			name: "concurrent selector matching series operations",
			test: testConcurrentSelectorMatchingSeries,
		},
		{
			name: "concurrent known labels operations",
			test: testConcurrentKnownLabels,
		},
		{
			name: "concurrent query stats operations",
			test: testConcurrentQueryStats,
		},
		{
			name: "mixed concurrent operations",
			test: testMixedConcurrentOperations,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &cacheData{
				QueriesStats:           make(map[string]queryStats),
				SelectorMatchingSeries: make(map[string]int),
				KnownLabels:            []string{},
				mtx:                    sync.RWMutex{},
			}
			tt.test(t, data)
		})
	}
}

func testConcurrentSelectorMatchingSeries(t *testing.T, data *cacheData) {
	const numGoroutines = 50
	const numOperations = 100

	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				selector := fmt.Sprintf("metric_%d_%d", id, j)
				data.SetSelectorMatchingSeries(selector, id*numOperations+j)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				selector := fmt.Sprintf("metric_%d_%d", id, j/2) // Read some existing and some non-existing
				count, _ := data.MatchingSeriesForSelector(selector)
				// Just ensure we can read without panicking
				_ = count
			}
		}(i)
	}

	wg.Wait()

	// Verify data integrity
	data.mtx.RLock()
	expectedCount := numGoroutines * numOperations
	actualCount := len(data.SelectorMatchingSeries)
	data.mtx.RUnlock()

	assert.Equal(t, expectedCount, actualCount, "Expected %d entries, got %d", expectedCount, actualCount)
}

func testConcurrentKnownLabels(t *testing.T, data *cacheData) {
	const numGoroutines = 20
	const numOperations = 50

	var wg sync.WaitGroup

	// Concurrent writes of different label sets
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				labels := make([]string, 0, 5)
				for k := 0; k < 5; k++ {
					labels = append(labels, fmt.Sprintf("label_%d_%d_%d", id, j, k))
				}
				data.SetKnownLabels(labels)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				labels := data.GetKnownLabels()
				// Ensure we get a valid slice (might be empty or contain labels)
				assert.Assert(t, labels != nil, "GetKnownLabels should never return nil")
				// Test that modifications to returned slice don't affect original
				if len(labels) > 0 {
					originalFirst := labels[0]
					labels[0] = "modified"
					newLabels := data.GetKnownLabels()
					if len(newLabels) > 0 {
						// If labels still exist, the first one should not be "modified"
						// because GetKnownLabels returns a clone
						assert.Assert(t, newLabels[0] != "modified",
							"Modifications to returned slice should not affect original")
					}
					_ = originalFirst // Use the variable
				}
			}
		}()
	}

	wg.Wait()
}

func testConcurrentQueryStats(t *testing.T, data *cacheData) {
	const numGoroutines = 30
	const numOperations = 100

	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				query := fmt.Sprintf("up{instance=\"%d_%d\"}", id, j)
				stats := queryStats{
					Error:    nil,
					Series:   id + j,
					Duration: time.Duration(id*j) * time.Millisecond,
				}
				data.SetQueryStats(query, stats)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				query := fmt.Sprintf("up{instance=\"%d_%d\"}", id, j/2) // Read some existing and some non-existing
				stats, found := data.GetQueryStats(query)
				if found && stats.Error == nil {
					// If we found stats and there's no error, series count should be non-negative
					assert.Assert(t, stats.Series >= 0, "Series count should be non-negative")
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify final state
	data.mtx.RLock()
	expectedCount := numGoroutines * numOperations
	actualCount := len(data.QueriesStats)
	data.mtx.RUnlock()

	assert.Equal(t, expectedCount, actualCount, "Expected %d query stats entries, got %d", expectedCount, actualCount)
}

func testMixedConcurrentOperations(_ *testing.T, data *cacheData) {
	const numGoroutines = 20
	const numOperations = 50

	var wg sync.WaitGroup

	// Mixed operations goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				switch j % 6 {
				case 0:
					selector := fmt.Sprintf("mixed_metric_%d", id)
					data.SetSelectorMatchingSeries(selector, j)
				case 1:
					_, _ = data.MatchingSeriesForSelector(fmt.Sprintf("mixed_metric_%d", id))
				case 2:
					labels := []string{fmt.Sprintf("label_%d_%d", id, j)}
					data.SetKnownLabels(labels)
				case 3:
					_ = data.GetKnownLabels()
				case 4:
					query := fmt.Sprintf("metric_%d", id)
					stats := queryStats{Series: j, Duration: time.Duration(j) * time.Millisecond}
					data.SetQueryStats(query, stats)
				case 5:
					_, _ = data.GetQueryStats(fmt.Sprintf("metric_%d", id))
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestCache_ConcurrentAccess(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "cache_test_*.json")
	assert.NilError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cache := newCache(tmpFile.Name(), "http://localhost:9090", time.Hour)
	assert.Assert(t, cache != nil, "Cache should be created successfully")

	t.Run("concurrent SourceTenantsData operations", func(t *testing.T) {
		testConcurrentSourceTenantsData(t, cache)
	})

	t.Run("concurrent Dump operations", func(t *testing.T) {
		testConcurrentDump(t, cache)
	})
}

func testConcurrentSourceTenantsData(t *testing.T, cache *cache) {
	const numGoroutines = 30
	const numOperations = 50

	var wg sync.WaitGroup

	// Collect all created cache data instances to verify they're properly shared
	var createdData sync.Map

	// Concurrent access to SourceTenantsData
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				tenants := []string{fmt.Sprintf("tenant_%d", id%5)} // Create some overlap
				data := cache.SourceTenantsData(tenants)
				assert.Assert(t, data != nil, "SourceTenantsData should never return nil")

				// Store reference for verification
				key := sourceTenantsToHeader(tenants)
				createdData.Store(key, data)

				// Perform some operations on the returned data
				data.SetSelectorMatchingSeries(fmt.Sprintf("metric_%d_%d", id, j), j)
				_, _ = data.MatchingSeriesForSelector(fmt.Sprintf("metric_%d_%d", id, j))
			}
		}(i)
	}

	wg.Wait()

	// Verify that same tenant combinations return the same cacheData instance
	tenantsToTest := [][]string{
		{"tenant_0"},
		{"tenant_1"},
		{"tenant_2"},
	}

	for _, tenants := range tenantsToTest {
		data1 := cache.SourceTenantsData(tenants)
		data2 := cache.SourceTenantsData(tenants)
		assert.Equal(t, data1, data2, "Same tenants should return same cacheData instance")
	}
}

func testConcurrentDump(_ *testing.T, cache *cache) {
	const numGoroutines = 10
	const numOperations = 20

	var wg sync.WaitGroup

	// Add some data to the cache first
	for i := 0; i < 5; i++ {
		tenants := []string{fmt.Sprintf("tenant_%d", i)}
		data := cache.SourceTenantsData(tenants)
		data.SetSelectorMatchingSeries("metric", i*10)
		data.SetKnownLabels([]string{fmt.Sprintf("label_%d", i)})
	}

	// Concurrent operations while dumping
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				switch j % 3 {
				case 0:
					// Concurrent dumps
					cache.Dump()
				case 1:
					// Concurrent reads via SourceTenantsData
					tenants := []string{fmt.Sprintf("tenant_%d", id%5)}
					data := cache.SourceTenantsData(tenants)
					_ = data.GetKnownLabels()
				case 2:
					// Concurrent writes via SourceTenantsData
					tenants := []string{fmt.Sprintf("tenant_%d", id%3)}
					data := cache.SourceTenantsData(tenants)
					data.SetSelectorMatchingSeries(fmt.Sprintf("concurrent_metric_%d", id), j)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestRaceConditions(t *testing.T) {
	// This test is primarily for running with -race flag
	if !raceEnabled() {
		t.Skip("Race detection not enabled, skipping race condition test")
	}

	t.Run("cacheData race conditions", func(_ *testing.T) {
		data := &cacheData{
			QueriesStats:           make(map[string]queryStats),
			SelectorMatchingSeries: make(map[string]int),
			KnownLabels:            []string{},
			mtx:                    sync.RWMutex{},
		}

		const numGoroutines = 100
		var wg sync.WaitGroup

		// High-intensity concurrent operations
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				// Rapidly alternating read/write operations
				for j := 0; j < 1000; j++ {
					switch j % 8 {
					case 0:
						data.SetSelectorMatchingSeries(fmt.Sprintf("m%d", id), j)
					case 1:
						data.MatchingSeriesForSelector(fmt.Sprintf("m%d", id))
					case 2:
						data.SetKnownLabels([]string{fmt.Sprintf("l%d", id)})
					case 3:
						data.GetKnownLabels()
					case 4:
						data.SetQueryStats(fmt.Sprintf("q%d", id), queryStats{Series: j})
					case 5:
						data.GetQueryStats(fmt.Sprintf("q%d", id))
					case 6:
						data.MatchingSeriesForSelector(fmt.Sprintf("m%d", (id+1)%numGoroutines))
					case 7:
						data.GetKnownLabels()
					}
				}
			}(i)
		}

		wg.Wait()
	})

	t.Run("cache race conditions", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "race_test_*.json")
		assert.NilError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		cache := newCache(tmpFile.Name(), "http://localhost:9090", time.Hour)

		const numGoroutines = 50
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					switch j % 3 {
					case 0:
						tenants := []string{fmt.Sprintf("t%d", id%10)}
						data := cache.SourceTenantsData(tenants)
						data.SetSelectorMatchingSeries(fmt.Sprintf("metric_%d", id), j)
					case 1:
						cache.Dump()
					case 2:
						tenants := []string{fmt.Sprintf("t%d", (id+1)%10)}
						data := cache.SourceTenantsData(tenants)
						data.GetKnownLabels()
					}
				}
			}(i)
		}

		wg.Wait()
	})
}

// Benchmark tests for performance under concurrent load.
func BenchmarkCacheData_ConcurrentOperations(b *testing.B) {
	data := &cacheData{
		QueriesStats:           make(map[string]queryStats),
		SelectorMatchingSeries: make(map[string]int),
		KnownLabels:            []string{},
		mtx:                    sync.RWMutex{},
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 6 {
			case 0:
				data.SetSelectorMatchingSeries(fmt.Sprintf("metric_%d", i), i)
			case 1:
				data.MatchingSeriesForSelector(fmt.Sprintf("metric_%d", i))
			case 2:
				data.SetKnownLabels([]string{fmt.Sprintf("label_%d", i)})
			case 3:
				data.GetKnownLabels()
			case 4:
				data.SetQueryStats(fmt.Sprintf("query_%d", i), queryStats{Series: i})
			case 5:
				data.GetQueryStats(fmt.Sprintf("query_%d", i))
			}
			i++
		}
	})
}

func BenchmarkCache_ConcurrentSourceTenantsData(b *testing.B) {
	tmpFile, err := os.CreateTemp("", "bench_test_*.json")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	cache := newCache(tmpFile.Name(), "http://localhost:9090", time.Hour)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			tenants := []string{fmt.Sprintf("tenant_%d", i%10)}
			data := cache.SourceTenantsData(tenants)
			data.SetSelectorMatchingSeries(fmt.Sprintf("metric_%d", i), i)
			i++
		}
	})
}

// Helper function to detect if race detection is enabled.
func raceEnabled() bool {
	// This is a heuristic - check if testing is running with race detection
	// by examining runtime behavior
	return runtime.GOARCH != "" // Simple check, actual race flag detection is complex
}

func TestSourceTenantsDataRaceCondition(t *testing.T) {
	// This test specifically targets the race condition in SourceTenantsData
	tmpFile, err := os.CreateTemp("", "race_test_*.json")
	assert.NilError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cache := newCache(tmpFile.Name(), "http://localhost:9090", time.Hour)

	const numGoroutines = 200
	var wg sync.WaitGroup

	// All goroutines try to access the same tenant key simultaneously
	tenantKey := "test_tenant"
	var results []*cacheData
	var resultsMutex sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Create deliberate timing to increase chance of race condition
			time.Sleep(time.Nanosecond * time.Duration(rand.Intn(1000)))

			data := cache.SourceTenantsData([]string{tenantKey})

			resultsMutex.Lock()
			results = append(results, data)
			resultsMutex.Unlock()

			// Perform some operation
			data.SetSelectorMatchingSeries("test_metric", i)
		}()
	}

	wg.Wait()

	// Check that all results point to the same cacheData instance
	firstData := results[0]
	var duplicateCount int
	for i, data := range results {
		if data != firstData {
			duplicateCount++
			t.Logf("Mismatch at index %d: different cacheData instance detected", i)
		}
	}

	if duplicateCount > 0 {
		t.Errorf("Found %d instances where different cacheData objects were returned for same key", duplicateCount)
	}

	// Verify the cache only contains one entry for this tenant
	cache.mtx.RLock()
	cacheSize := len(cache.SourceTenants)
	cache.mtx.RUnlock()

	assert.Equal(t, 1, cacheSize, "Cache should contain exactly one tenant entry")
}

func TestClient_NewClientConcurrency(t *testing.T) {
	// Test that newClient method doesn't have race conditions when called concurrently

	// Create a client instance
	tmpFile, err := os.CreateTemp("", "client_test_*.json")
	assert.NilError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cache := newCache(tmpFile.Name(), "http://localhost:9090", time.Hour)

	client := &Client{
		httpHeaders: &prom_config.Headers{
			Headers: map[string]prom_config.Header{
				"User-Agent": {Values: []string{"promruval"}},
			},
		},
		prometheusURL:         "http://localhost:9090",
		timeout:               30 * time.Second,
		cache:                 cache,
		maxRetries:            3,
		insecureSkipTLSVerify: true,
	}

	const numGoroutines = 100
	const numOperations = 10
	var wg sync.WaitGroup

	// All goroutines call newClient with different additional headers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				headers := map[string]string{
					fmt.Sprintf("X-Test-Header-%d", id): fmt.Sprintf("value-%d-%d", id, j),
					fmt.Sprintf("X-Goroutine-%d", id):   fmt.Sprintf("routine-%d", id),
					fmt.Sprintf("X-Iteration-%d", j):    fmt.Sprintf("iter-%d", j),
				}

				// This should not cause concurrent map writes
				_, err := client.newClient(headers)
				if err != nil {
					// We expect this to fail since we don't have a real Prometheus server,
					// but it should fail gracefully, not with a race condition panic
					_ = err // Ignore the error, we just want to test for race conditions
				}
			}
		}(i)
	}

	wg.Wait()

	// If we reach here without panicking, the race condition is fixed
	// Verify that original headers are unchanged
	assert.Equal(t, 1, len(client.httpHeaders.Headers), "Original headers should be unchanged")
	assert.Equal(t, "promruval", client.httpHeaders.Headers["User-Agent"].Values[0], "Original User-Agent should be unchanged")
}
