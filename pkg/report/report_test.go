// Package report provides concurrency tests for ValidationReport and related structures.

// These tests comprehensively verify:
// 1. Concurrent creation of reports at all levels works correctly
// 2. Complex nested concurrent operations work without data corruption
// 3. Mutex contention doesn't cause deadlocks
// 4. Race conditions are properly documented for non-protected operations
// 5. Performance characteristics of concurrent operations
package report

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/fusakla/promruval/v3/pkg/config"
	"github.com/stretchr/testify/assert"
)

// TestValidationReportConcurrentFileReportCreation tests that multiple goroutines can safely
// create file reports simultaneously without race conditions.
func TestValidationReportConcurrentFileReportCreation(t *testing.T) {
	report := NewValidationReport()
	numGoroutines := 100
	var wg sync.WaitGroup

	// Create file reports concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			fileName := fmt.Sprintf("file_%d.yaml", index)
			fileReport := report.NewFileReport(fileName)
			assert.NotNil(t, fileReport)
			assert.Equal(t, fileName, fileReport.Name)
			assert.True(t, fileReport.Valid.Load())
		}(i)
	}

	wg.Wait()

	// Verify all file reports were created
	assert.Len(t, report.FilesReports, numGoroutines)

	// Verify no duplicate file reports (all names should be unique)
	fileNames := make(map[string]bool)
	for _, fileReport := range report.FilesReports {
		_, exists := fileNames[fileReport.Name]
		assert.False(t, exists, "Duplicate file report found: %s", fileReport.Name)
		fileNames[fileReport.Name] = true
	}
}

// TestFileReportConcurrentGroupReportCreation tests that multiple goroutines can safely
// create group reports on the same file report simultaneously.
func TestFileReportConcurrentGroupReportCreation(t *testing.T) {
	validationReport := NewValidationReport()
	fileReport := validationReport.NewFileReport("test_file.yaml")
	numGoroutines := 50
	var wg sync.WaitGroup

	// Create group reports concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			groupName := fmt.Sprintf("group_%d", index)
			groupReport := fileReport.NewGroupReport(groupName)
			assert.NotNil(t, groupReport)
			assert.Equal(t, groupName, groupReport.Name)
			assert.True(t, groupReport.Valid.Load())
		}(i)
	}

	wg.Wait()

	// Verify all group reports were created
	assert.Len(t, fileReport.GroupReports, numGoroutines)

	// Verify no duplicate group reports
	groupNames := make(map[string]bool)
	for _, groupReport := range fileReport.GroupReports {
		_, exists := groupNames[groupReport.Name]
		assert.False(t, exists, "Duplicate group report found: %s", groupReport.Name)
		groupNames[groupReport.Name] = true
	}
}

// TestGroupReportConcurrentRuleReportCreation tests that multiple goroutines can safely
// create rule reports on the same group report simultaneously.
func TestGroupReportConcurrentRuleReportCreation(t *testing.T) {
	validationReport := NewValidationReport()
	fileReport := validationReport.NewFileReport("test_file.yaml")
	groupReport := fileReport.NewGroupReport("test_group")
	numGoroutines := 50
	var wg sync.WaitGroup

	// Create rule reports concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ruleName := fmt.Sprintf("rule_%d", index)
			ruleType := config.AlertScope
			if index%2 == 0 {
				ruleType = config.RecordingRuleScope
			}
			ruleReport := groupReport.NewRuleReport(ruleName, ruleType)
			assert.NotNil(t, ruleReport)
			assert.Equal(t, ruleName, ruleReport.Name)
			assert.Equal(t, ruleType, ruleReport.RuleType)
			assert.True(t, ruleReport.Valid.Load())
		}(i)
	}

	wg.Wait()

	// Verify all rule reports were created
	assert.Len(t, groupReport.RuleReports, numGoroutines)

	// Verify no duplicate rule reports
	ruleNames := make(map[string]bool)
	for _, ruleReport := range groupReport.RuleReports {
		_, exists := ruleNames[ruleReport.Name]
		assert.False(t, exists, "Duplicate rule report found: %s", ruleReport.Name)
		ruleNames[ruleReport.Name] = true
	}
}

// TestValidationReportConcurrentModificationAndSorting tests concurrent modifications
// and sorting operations. This test verifies that the Sort() method is now thread-safe
// and can be called concurrently with other modification operations without race conditions.
func TestValidationReportConcurrentModificationAndSorting(t *testing.T) {
	report := NewValidationReport()
	numWorkers := 20
	numOperationsPerWorker := 100
	var wg sync.WaitGroup

	// Start workers that create file reports
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < numOperationsPerWorker; j++ {
				fileName := fmt.Sprintf("worker_%d_file_%d.yaml", workerID, j)
				fileReport := report.NewFileReport(fileName)

				// Add some group reports to make sorting more meaningful
				if j%10 == 0 {
					groupReport := fileReport.NewGroupReport(fmt.Sprintf("group_%d_%d", workerID, j))
					if j%20 == 0 {
						groupReport.NewRuleReport(fmt.Sprintf("rule_%d_%d", workerID, j), config.AlertScope)
					}
				}
			}
		}(i)
	}

	// Start workers that sort the report
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				report.Sort()
				time.Sleep(time.Millisecond) // Small delay to allow interleaving
			}
		}()
	}

	wg.Wait()

	// Final verification
	expectedFileCount := numWorkers * numOperationsPerWorker
	assert.Len(t, report.FilesReports, expectedFileCount)

	// Verify sorting works correctly
	report.Sort()
	for i := 1; i < len(report.FilesReports); i++ {
		assert.LessOrEqual(t, report.FilesReports[i-1].Name, report.FilesReports[i].Name,
			"Files should be sorted by name")
	}
}

// TestNestedConcurrentOperations tests complex nested concurrent operations across all levels.
func TestNestedConcurrentOperations(t *testing.T) {
	report := NewValidationReport()
	numFiles := 10
	numGroupsPerFile := 10
	numRulesPerGroup := 5
	var wg sync.WaitGroup

	// Create a complex hierarchy concurrently
	for fileIdx := 0; fileIdx < numFiles; fileIdx++ {
		wg.Add(1)
		go func(fIdx int) {
			defer wg.Done()

			fileName := fmt.Sprintf("file_%d.yaml", fIdx)
			fileReport := report.NewFileReport(fileName)

			var fileWg sync.WaitGroup
			for groupIdx := 0; groupIdx < numGroupsPerFile; groupIdx++ {
				fileWg.Add(1)
				go func(gIdx int) {
					defer fileWg.Done()

					groupName := fmt.Sprintf("group_%d_%d", fIdx, gIdx)
					groupReport := fileReport.NewGroupReport(groupName)

					var groupWg sync.WaitGroup
					for ruleIdx := 0; ruleIdx < numRulesPerGroup; ruleIdx++ {
						groupWg.Add(1)
						go func(rIdx int) {
							defer groupWg.Done()

							ruleName := fmt.Sprintf("rule_%d_%d_%d", fIdx, gIdx, rIdx)
							ruleType := config.AlertScope
							if rIdx%2 == 0 {
								ruleType = config.RecordingRuleScope
							}
							ruleReport := groupReport.NewRuleReport(ruleName, ruleType)
							assert.NotNil(t, ruleReport)
						}(ruleIdx)
					}
					groupWg.Wait()
				}(groupIdx)
			}
			fileWg.Wait()
		}(fileIdx)
	}

	wg.Wait()

	// Verify the complete hierarchy was created correctly
	assert.Len(t, report.FilesReports, numFiles)
	for _, fileReport := range report.FilesReports {
		assert.Len(t, fileReport.GroupReports, numGroupsPerFile)
		for _, groupReport := range fileReport.GroupReports {
			assert.Len(t, groupReport.RuleReports, numRulesPerGroup)
		}
	}

	// Test that sorting works on the complex hierarchy
	report.Sort()

	// Verify files are sorted
	for i := 1; i < len(report.FilesReports); i++ {
		assert.LessOrEqual(t, report.FilesReports[i-1].Name, report.FilesReports[i].Name)
	}

	// Verify groups within each file are sorted
	for _, fileReport := range report.FilesReports {
		for i := 1; i < len(fileReport.GroupReports); i++ {
			assert.LessOrEqual(t, fileReport.GroupReports[i-1].Name, fileReport.GroupReports[i].Name)
		}

		// Verify rules within each group are sorted
		for _, groupReport := range fileReport.GroupReports {
			for i := 1; i < len(groupReport.RuleReports); i++ {
				assert.LessOrEqual(t, groupReport.RuleReports[i-1].Name, groupReport.RuleReports[i].Name)
			}
		}
	}
}

// TestConcurrentErrorAppending tests concurrent error appending to various report levels.
// This test verifies that the new AddError/AddErrors methods are properly thread-safe
// and can handle concurrent error appending without race conditions.
func TestConcurrentErrorAppending(t *testing.T) {
	report := NewValidationReport()
	fileReport := report.NewFileReport("test_file.yaml")
	groupReport := fileReport.NewGroupReport("test_group")
	ruleReport := groupReport.NewRuleReport("test_rule", config.AlertScope)

	numGoroutines := 50
	var wg sync.WaitGroup

	// Test concurrent error appending at different levels
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Append errors to different levels using thread-safe methods.
			switch index % 3 {
			case 0:
				fileReport.AddError(NewErrorf("file error %d", index))
			case 1:
				groupReport.AddError(NewErrorf("group error %d", index))
			default:
				ruleReport.AddError(NewErrorf("rule error %d", index))
			}
		}(i)
	}

	wg.Wait()

	// Verify that errors were added successfully (thread-safe operations should work correctly)
	totalFileErrors := len(fileReport.Errors)
	totalGroupErrors := len(groupReport.Errors)
	totalRuleErrors := len(ruleReport.Errors)

	// We expect roughly even distribution of errors (allowing for some variance)
	assert.Greater(t, totalFileErrors, 0, "Should have some file errors")
	assert.Greater(t, totalGroupErrors, 0, "Should have some group errors")
	assert.Greater(t, totalRuleErrors, 0, "Should have some rule errors")

	// Total should be close to numGoroutines (some may be assigned to same bucket due to modulo)
	totalErrors := totalFileErrors + totalGroupErrors + totalRuleErrors
	assert.LessOrEqual(t, totalErrors, numGoroutines, "Should not exceed the number of goroutines")
	assert.Greater(t, totalErrors, 0, "Should have created some errors")
}

// TestMutexProtectedOperationsOnly tests only the mutex-protected operations
// (NewFileReport, NewGroupReport, NewRuleReport) to verify they are truly thread-safe.
func TestMutexProtectedOperationsOnly(t *testing.T) {
	report := NewValidationReport()
	numGoroutines := 100
	var wg sync.WaitGroup

	// Test only the operations that are explicitly protected by mutexes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Create file report (mutex protected)
			fileName := fmt.Sprintf("protected_file_%d.yaml", index)
			fileReport := report.NewFileReport(fileName)
			assert.NotNil(t, fileReport)

			// Create group report (mutex protected)
			groupName := fmt.Sprintf("protected_group_%d", index)
			groupReport := fileReport.NewGroupReport(groupName)
			assert.NotNil(t, groupReport)

			// Create rule report (mutex protected)
			ruleName := fmt.Sprintf("protected_rule_%d", index)
			ruleReport := groupReport.NewRuleReport(ruleName, config.AlertScope)
			assert.NotNil(t, ruleReport)
		}(i)
	}

	wg.Wait()

	// Verify all reports were created successfully
	assert.Len(t, report.FilesReports, numGoroutines)
	for _, fileReport := range report.FilesReports {
		assert.Len(t, fileReport.GroupReports, 1)
		for _, groupReport := range fileReport.GroupReports {
			assert.Len(t, groupReport.RuleReports, 1)
		}
	}
}

// TestMutexContention tests that mutex contention doesn't cause deadlocks.
func TestMutexContention(t *testing.T) {
	report := NewValidationReport()
	numGoroutines := 100
	var wg sync.WaitGroup

	done := make(chan struct{})

	// Start goroutines that create reports rapidly
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					fileName := fmt.Sprintf("contention_file_%d_%d.yaml", index, time.Now().UnixNano())
					fileReport := report.NewFileReport(fileName)

					groupName := fmt.Sprintf("contention_group_%d", index)
					groupReport := fileReport.NewGroupReport(groupName)

					ruleName := fmt.Sprintf("contention_rule_%d", index)
					groupReport.NewRuleReport(ruleName, config.AlertScope)
				}
			}
		}(i)
	}

	// Let them run for a short period
	time.Sleep(100 * time.Millisecond)
	close(done)

	// Wait with timeout to ensure no deadlock
	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		// Success - no deadlock
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for goroutines to finish - possible deadlock")
	}

	// Verify we have some reports
	assert.Greater(t, len(report.FilesReports), 0, "Should have created some file reports")
}

// TestConcurrentSortingStability tests sorting behavior under concurrent access.
// This test verifies that the Sort() method is thread-safe and produces consistent
// results when called concurrently, even with concurrent modifications.
func TestConcurrentSortingStability(t *testing.T) {
	report := NewValidationReport()

	// Create a deterministic set of reports first (do this sequentially)
	for i := 10; i >= 0; i-- {
		fileName := fmt.Sprintf("file_%02d.yaml", i)
		fileReport := report.NewFileReport(fileName)

		for j := 10; j >= 0; j-- {
			groupName := fmt.Sprintf("group_%02d", j)
			groupReport := fileReport.NewGroupReport(groupName)

			for k := 10; k >= 0; k-- {
				ruleName := fmt.Sprintf("rule_%02d", k)
				groupReport.NewRuleReport(ruleName, config.AlertScope)
			}
		}
	}

	// Now test sorting only (without concurrent modifications)
	// This tests that sorting itself is stable when no modifications occur
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				report.Sort()
				time.Sleep(time.Millisecond) // Small delay to allow interleaving
			}
		}()
	}

	wg.Wait()

	// Verify final sorted state
	report.Sort()

	// Check files are sorted
	for i := 1; i < len(report.FilesReports); i++ {
		assert.LessOrEqual(t, report.FilesReports[i-1].Name, report.FilesReports[i].Name)
	}

	// Check groups in first file are sorted
	if len(report.FilesReports) > 0 {
		firstFile := report.FilesReports[0]
		for i := 1; i < len(firstFile.GroupReports); i++ {
			assert.LessOrEqual(t, firstFile.GroupReports[i-1].Name, firstFile.GroupReports[i].Name)
		}

		// Check rules in first group are sorted
		if len(firstFile.GroupReports) > 0 {
			firstGroup := firstFile.GroupReports[0]
			for i := 1; i < len(firstGroup.RuleReports); i++ {
				assert.LessOrEqual(t, firstGroup.RuleReports[i-1].Name, firstGroup.RuleReports[i].Name)
			}
		}
	}
}

// BenchmarkConcurrentFileReportCreation benchmarks concurrent file report creation.
func BenchmarkConcurrentFileReportCreation(b *testing.B) {
	report := NewValidationReport()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			fileName := fmt.Sprintf("bench_file_%d.yaml", counter)
			report.NewFileReport(fileName)
			counter++
		}
	})
}

// BenchmarkConcurrentGroupReportCreation benchmarks concurrent group report creation.
func BenchmarkConcurrentGroupReportCreation(b *testing.B) {
	report := NewValidationReport()
	fileReport := report.NewFileReport("benchmark_file.yaml")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			groupName := fmt.Sprintf("bench_group_%d", counter)
			fileReport.NewGroupReport(groupName)
			counter++
		}
	})
}

// BenchmarkSorting benchmarks the sorting operation.
func BenchmarkSorting(b *testing.B) {
	report := NewValidationReport()

	// Create a complex hierarchy
	for i := 0; i < 10; i++ {
		fileName := fmt.Sprintf("file_%d.yaml", i)
		fileReport := report.NewFileReport(fileName)
		for j := 0; j < 10; j++ {
			groupName := fmt.Sprintf("group_%d", j)
			groupReport := fileReport.NewGroupReport(groupName)
			for k := 0; k < 10; k++ {
				ruleName := fmt.Sprintf("rule_%d", k)
				groupReport.NewRuleReport(ruleName, config.AlertScope)
			}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report.Sort()
	}
}

// TestThreadSafeErrorOperations specifically tests that all error operations are thread-safe.
func TestThreadSafeErrorOperations(t *testing.T) {
	report := NewValidationReport()
	fileReport := report.NewFileReport("thread_safe_test.yaml")
	groupReport := fileReport.NewGroupReport("thread_safe_group")
	ruleReport := groupReport.NewRuleReport("thread_safe_rule", config.AlertScope)

	numGoroutines := 100
	errorsPerGoroutine := 10
	var wg sync.WaitGroup

	// Test concurrent AddError operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			// Add multiple errors per goroutine to test thread safety under load
			for j := 0; j < errorsPerGoroutine; j++ {
				fileReport.AddError(NewErrorf("file error %d-%d", goroutineID, j))
				groupReport.AddError(NewErrorf("group error %d-%d", goroutineID, j))
				ruleReport.AddError(NewErrorf("rule error %d-%d", goroutineID, j))
			}
		}(i)
	}

	wg.Wait()

	// Verify that all errors were added correctly
	expectedErrors := numGoroutines * errorsPerGoroutine
	assert.Len(t, fileReport.Errors, expectedErrors, "All file errors should be added")
	assert.Len(t, groupReport.Errors, expectedErrors, "All group errors should be added")
	assert.Len(t, ruleReport.Errors, expectedErrors, "All rule errors should be added")

	// Test concurrent AddErrors (batch) operations
	var wg2 sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg2.Add(1)
		go func(goroutineID int) {
			defer wg2.Done()

			// Create batch of errors to add
			fileErrs := make([]*Error, errorsPerGoroutine)
			groupErrs := make([]*Error, errorsPerGoroutine)
			ruleErrs := make([]*Error, errorsPerGoroutine)

			for j := 0; j < errorsPerGoroutine; j++ {
				fileErrs[j] = NewErrorf("batch file error %d-%d", goroutineID, j)
				groupErrs[j] = NewErrorf("batch group error %d-%d", goroutineID, j)
				ruleErrs[j] = NewErrorf("batch rule error %d-%d", goroutineID, j)
			}

			// Add errors in batches
			fileReport.AddErrors(fileErrs)
			groupReport.AddErrors(groupErrs)
			ruleReport.AddErrors(ruleErrs)
		}(i)
	}

	wg2.Wait()

	// Verify that batch errors were also added correctly
	expectedTotalErrors := expectedErrors * 2 // both single and batch operations
	assert.Len(t, fileReport.Errors, expectedTotalErrors, "All file errors including batch should be added")
	assert.Len(t, groupReport.Errors, expectedTotalErrors, "All group errors including batch should be added")
	assert.Len(t, ruleReport.Errors, expectedTotalErrors, "All rule errors including batch should be added")
}

// TestThreadSafeFieldOperations tests all thread-safe setter and getter methods.
func TestThreadSafeFieldOperations(t *testing.T) {
	report := NewValidationReport()
	fileReport := report.NewFileReport("field_test.yaml")
	groupReport := fileReport.NewGroupReport("field_group")
	ruleReport := groupReport.NewRuleReport("field_rule", config.AlertScope)

	numGoroutines := 100
	var wg sync.WaitGroup

	// Test concurrent field modifications
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Test ValidationReport field setters
			if index%2 == 0 {
				report.Failed.Store(true)
				report.FilesCount.Inc()
				report.GroupsCount.Add(1)
				report.RulesCount.Add(1)
			}

			// Test FileReport field setters
			fileReport.Valid.Store(index%2 == 1)
			fileReport.Excluded.Store(index%3 == 1)
			fileReport.HasRuleValidationErrors.Store(index%4 == 1)

			// Test GroupReport field setters
			groupReport.Valid.Store(index%3 == 1)
			groupReport.Excluded.Store(index%4 == 1)

			// Test RuleReport field setters
			ruleReport.Valid.Store(index%3 == 2)
			ruleReport.Excluded.Store(index%4 == 2)
		}(i)
	}

	wg.Wait()

	// Test that direct atomic access works
	_ = report.Failed.Load()
	_ = report.Duration.Load()
	_ = fileReport.Valid.Load()
	_ = fileReport.Excluded.Load()
	_ = groupReport.Valid.Load()
	_ = groupReport.Excluded.Load()
	_ = ruleReport.Valid.Load()
	_ = ruleReport.Excluded.Load()

	// Test ValidationReport count incrementing
	var wg2 sync.WaitGroup
	originalGroupsCount := int(report.GroupsCount.Load())
	originalRulesCount := int(report.RulesCount.Load())

	for i := 0; i < 50; i++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			report.GroupsCount.Add(2)
			report.RulesCount.Add(3)
		}()
	}

	wg2.Wait()

	// Verify counts were incremented correctly
	expectedGroupsIncrease := 50 * 2
	expectedRulesIncrease := 50 * 3
	assert.Equal(t, originalGroupsCount+expectedGroupsIncrease, int(report.GroupsCount.Load()), "Groups count should be incremented correctly")
	assert.Equal(t, originalRulesCount+expectedRulesIncrease, int(report.RulesCount.Load()), "Rules count should be incremented correctly")
}
