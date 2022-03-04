package relay

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPairedRelayEnableFor(t *testing.T) {
	first, second := NewRelayPair(&Dumb{}, &Dumb{})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	t.Run("second relay will be not enabled until first gets released", func(t *testing.T) {
		start := time.Now()
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			first.EnableFor(ctx, time.Millisecond*5)
			wg.Done()
		}()

		wg.Add(1)
		go func() {
			second.EnableFor(ctx, time.Millisecond*5)
			wg.Done()
		}()

		wg.Wait()
		assert.GreaterOrEqual(t, time.Since(start), time.Millisecond*10)
	})
}
