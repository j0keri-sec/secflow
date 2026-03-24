package pusher

import (
	"context"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
)

// Multi implements multi-channel push with error aggregation.
type Multi struct {
	pushers  []TextPusher
	interval time.Duration
}

// NewMulti creates a new multi-channel pusher.
func NewMulti(pushers ...TextPusher) *Multi {
	return &Multi{
		pushers: pushers,
	}
}

// WithInterval sets the interval between pushes to different channels.
func (m *Multi) WithInterval(interval time.Duration) *Multi {
	m.interval = interval
	return m
}

// PushText sends text to all channels.
func (m *Multi) PushText(ctx context.Context, text string) error {
	var errs *multierror.Error
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, pusher := range m.pushers {
		wg.Add(1)
		go func(p TextPusher) {
			defer wg.Done()
			if err := p.PushText(ctx, text); err != nil {
				mu.Lock()
				errs = multierror.Append(errs, err)
				mu.Unlock()
			}
		}(pusher)

		if m.interval > 0 {
			time.Sleep(m.interval)
		}
	}

	wg.Wait()
	return errs.ErrorOrNil()
}

// PushMarkdown sends markdown to all channels.
func (m *Multi) PushMarkdown(ctx context.Context, title, content string) error {
	var errs *multierror.Error
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, pusher := range m.pushers {
		wg.Add(1)
		go func(p TextPusher) {
			defer wg.Done()
			if err := p.PushMarkdown(ctx, title, content); err != nil {
				mu.Lock()
				errs = multierror.Append(errs, err)
				mu.Unlock()
			}
		}(pusher)

		if m.interval > 0 {
			time.Sleep(m.interval)
		}
	}

	wg.Wait()
	return errs.ErrorOrNil()
}

// SequentialMulti implements sequential multi-channel push.
// This is useful when you want to ensure order or avoid rate limits.
type SequentialMulti struct {
	pushers  []TextPusher
	interval time.Duration
}

// NewSequentialMulti creates a new sequential multi-channel pusher.
func NewSequentialMulti(pushers ...TextPusher) *SequentialMulti {
	return &SequentialMulti{
		pushers: pushers,
	}
}

// WithInterval sets the interval between pushes to different channels.
func (s *SequentialMulti) WithInterval(interval time.Duration) *SequentialMulti {
	s.interval = interval
	return s
}

// PushText sends text to all channels sequentially.
func (s *SequentialMulti) PushText(ctx context.Context, text string) error {
	var errs *multierror.Error

	for _, pusher := range s.pushers {
		if err := pusher.PushText(ctx, text); err != nil {
			errs = multierror.Append(errs, err)
		}

		if s.interval > 0 {
			time.Sleep(s.interval)
		}
	}

	return errs.ErrorOrNil()
}

// PushMarkdown sends markdown to all channels sequentially.
func (s *SequentialMulti) PushMarkdown(ctx context.Context, title, content string) error {
	var errs *multierror.Error

	for _, pusher := range s.pushers {
		if err := pusher.PushMarkdown(ctx, title, content); err != nil {
			errs = multierror.Append(errs, err)
		}

		if s.interval > 0 {
			time.Sleep(s.interval)
		}
	}

	return errs.ErrorOrNil()
}
