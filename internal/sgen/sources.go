package sgen

import "context"

type Supplier interface {
	ShouldCache() bool
	Supply(context.Context) ([]map[string]string, error)
}

type Source struct {
	Name      string
	Supplier  Supplier
	Renderers map[string]Renderer
}

func (s *Source) Load(ctx context.Context) ([]map[string]string, error) {
	if !s.Supplier.ShouldCache() {
		return s.Supplier.Supply(ctx)
	}

	cache, err := NewSourceCache()
	if err != nil {
		return nil, err
	}

	return cache.Load(s.Name)
}

// Sync updates the source's cache with the latest values from it's supplier.
func (s *Source) Sync(ctx context.Context) error {
	if !s.Supplier.ShouldCache() {
		return nil
	}

	data, err := s.Supplier.Supply(ctx)
	if err != nil {
		return err
	}

	cache, err := NewSourceCache()
	if err != nil {
		return err
	}

	return cache.Store(s.Name, data)
}
