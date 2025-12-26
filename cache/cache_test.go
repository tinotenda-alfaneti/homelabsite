package cache

import (
	"testing"
	"time"
)

const testComputedValue = "computed-value"

func TestCacheSetGet(t *testing.T) {
	c := New()

	// Test basic set/get
	c.Set("key1", "value1", 0)
	val, found := c.Get("key1")
	if !found {
		t.Error("Expected to find key1")
	}
	if val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}
}

func TestCacheExpiration(t *testing.T) {
	c := New()

	// Set item with short expiration
	c.Set("key2", "value2", 100*time.Millisecond)

	// Should exist immediately
	val, found := c.Get("key2")
	if !found {
		t.Error("Expected to find key2")
	}
	if val != "value2" {
		t.Errorf("Expected value2, got %v", val)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, found = c.Get("key2")
	if found {
		t.Error("Expected key2 to be expired")
	}
}

func TestCacheDelete(t *testing.T) {
	c := New()

	c.Set("key3", "value3", 0)
	c.Delete("key3")

	_, found := c.Get("key3")
	if found {
		t.Error("Expected key3 to be deleted")
	}
}

func TestCacheClear(t *testing.T) {
	c := New()

	c.Set("key4", "value4", 0)
	c.Set("key5", "value5", 0)
	c.Clear()

	_, found1 := c.Get("key4")
	_, found2 := c.Get("key5")

	if found1 || found2 {
		t.Error("Expected all keys to be cleared")
	}
}

func TestCacheGetOrSet(t *testing.T) {
	c := New()

	callCount := 0
	getValue := func() (interface{}, error) {
		callCount++
		return testComputedValue, nil
	}

	// First call should execute function
	val1, err := c.GetOrSet("key6", time.Minute, getValue)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if val1 != "computed-value" {
		t.Errorf("Expected computed-value, got %v", val1)
	}
	if callCount != 1 {
		t.Errorf("Expected function to be called once, got %d", callCount)
	}

	// Second call should use cached value
	val2, err := c.GetOrSet("key6", time.Minute, getValue)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if val2 != "computed-value" {
		t.Errorf("Expected computed-value, got %v", val2)
	}
	if callCount != 1 {
		t.Errorf("Expected function to still be called once, got %d", callCount)
	}
}

func TestCacheNoExpiration(t *testing.T) {
	c := New()

	// Set item without expiration
	c.Set("key7", "value7", 0)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Should still exist
	val, found := c.Get("key7")
	if !found {
		t.Error("Expected to find key7")
	}
	if val != "value7" {
		t.Errorf("Expected value7, got %v", val)
	}
}

func TestCacheConcurrent(t *testing.T) {
	c := New()

	// Test concurrent writes and reads
	done := make(chan bool)

	// Writer goroutines
	for i := 0; i < 10; i++ {
		go func(i int) {
			key := "concurrent-key"
			c.Set(key, i, 0)
			done <- true
		}(i)
	}

	// Reader goroutines
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = c.Get("concurrent-key")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Should have a value (race-free)
	_, found := c.Get("concurrent-key")
	if !found {
		t.Error("Expected to find concurrent-key")
	}
}
