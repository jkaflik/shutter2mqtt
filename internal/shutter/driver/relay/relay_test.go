package relay

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPoolEnableFor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	pool := make(chan struct{}, 4)

	t.Run("2 relays will run at once on a pool of 4", func(t *testing.T) {
		start := time.Now()
		enableProxiedRelaysFor(ctx, pool, 4, time.Millisecond*5)
		assert.GreaterOrEqual(t, time.Since(start), time.Millisecond*5)
	})

	t.Run("4 relays will run at once on a pool of 4", func(t *testing.T) {
		start := time.Now()
		enableProxiedRelaysFor(ctx, pool, 4, time.Millisecond*5)
		assert.GreaterOrEqual(t, time.Since(start), time.Millisecond*5)
	})

	t.Run("6 relays will run in two batches on a pool of 4", func(t *testing.T) {
		start := time.Now()
		enableProxiedRelaysFor(ctx, pool, 6, time.Millisecond*5)
		assert.GreaterOrEqual(t, time.Since(start), time.Millisecond*10)
	})

	t.Run("9 relays will run in three batches on a pool of 4", func(t *testing.T) {
		start := time.Now()
		enableProxiedRelaysFor(ctx, pool, 9, time.Millisecond*5)
		assert.GreaterOrEqual(t, time.Since(start), time.Millisecond*15)
	})
}

func enableProxiedRelaysFor(ctx context.Context, pool chan struct{}, num int, duration time.Duration) {
	var wg sync.WaitGroup

	for i := 0; i < num; i++ {
		relay := NewPoolProxy(&Dumb{}, pool)
		wg.Add(1)
		go func() {
			relay.EnableFor(ctx, duration)
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestDumbEnableFor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	relay := Dumb{}

	t.Run("relay enabled for 5ms will be executed at least 5ms", func(t *testing.T) {
		expectedDuration := time.Millisecond * 5
		start := time.Now()
		relay.EnableFor(ctx, expectedDuration)
		assert.GreaterOrEqual(t, time.Since(start), expectedDuration)
	})
}
