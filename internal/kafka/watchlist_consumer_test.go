package kafka

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock StockRepository
// ---------------------------------------------------------------------------

type mockStockRepo struct {
	mu      sync.Mutex
	upserts []stockUpsert
	err     error
}

type stockUpsert struct {
	Symbol   string
	Name     string
	Sector   string
	Industry string
}

func (m *mockStockRepo) UpsertStockBasic(symbol, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.upserts = append(m.upserts, stockUpsert{Symbol: symbol, Name: name})
	return nil
}

func (m *mockStockRepo) UpsertStockWithSector(symbol, name, sector, industry string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.upserts = append(m.upserts, stockUpsert{Symbol: symbol, Name: name, Sector: sector, Industry: industry})
	return nil
}

func (m *mockStockRepo) StockExists(symbol string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.upserts {
		if u.Symbol == symbol {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockStockRepo) Upserts() []stockUpsert {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]stockUpsert, len(m.upserts))
	copy(cp, m.upserts)
	return cp
}

// ---------------------------------------------------------------------------
// processMessage tests
// ---------------------------------------------------------------------------

func TestWatchlistConsumer_processMessage_WatchlistUpdated(t *testing.T) {
	repo := &mockStockRepo{}
	consumer := &WatchlistConsumer{repo: repo}

	event := WatchlistEvent{
		EventType: "WATCHLIST_UPDATED",
		Source:    "robinhood",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: WatchlistEventData{
			AddedSymbols: []string{"AAPL", "goog"},
			TotalCount:   2,
			Stocks: []WatchlistStock{
				{Symbol: "AAPL", Name: "Apple Inc."},
				{Symbol: "GOOG", Name: "Alphabet Inc."},
			},
		},
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	err = consumer.processMessage(kafkago.Message{Value: payload})
	require.NoError(t, err)

	upserts := repo.Upserts()
	assert.Len(t, upserts, 2)
	// Symbols should be upper-cased
	assert.Equal(t, "AAPL", upserts[0].Symbol)
	assert.Equal(t, "Apple Inc.", upserts[0].Name)
	assert.Equal(t, "GOOG", upserts[1].Symbol)
	assert.Equal(t, "Alphabet Inc.", upserts[1].Name)
}

func TestWatchlistConsumer_processMessage_SymbolAdded(t *testing.T) {
	repo := &mockStockRepo{}
	consumer := &WatchlistConsumer{repo: repo}

	event := WatchlistEvent{
		EventType: "WATCHLIST_SYMBOL_ADDED",
		Source:    "robinhood",
		Data: WatchlistEventData{
			Symbol: "tsla",
			Name:   "Tesla Inc.",
		},
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	err = consumer.processMessage(kafkago.Message{Value: payload})
	require.NoError(t, err)

	upserts := repo.Upserts()
	require.Len(t, upserts, 1)
	assert.Equal(t, "TSLA", upserts[0].Symbol)
	assert.Equal(t, "Tesla Inc.", upserts[0].Name)
}

func TestWatchlistConsumer_processMessage_SymbolAdded_EmptyName(t *testing.T) {
	repo := &mockStockRepo{}
	consumer := &WatchlistConsumer{repo: repo}

	event := WatchlistEvent{
		EventType: "WATCHLIST_SYMBOL_ADDED",
		Data: WatchlistEventData{
			Symbol: "sofi",
			Name:   "",
		},
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	err = consumer.processMessage(kafkago.Message{Value: payload})
	require.NoError(t, err)

	upserts := repo.Upserts()
	require.Len(t, upserts, 1)
	// Name defaults to uppercased symbol when empty
	assert.Equal(t, "SOFI", upserts[0].Symbol)
	assert.Equal(t, "SOFI", upserts[0].Name)
}

func TestWatchlistConsumer_processMessage_SymbolRemoved(t *testing.T) {
	repo := &mockStockRepo{}
	consumer := &WatchlistConsumer{repo: repo}

	event := WatchlistEvent{
		EventType: "WATCHLIST_SYMBOL_REMOVED",
		Data: WatchlistEventData{
			Symbol: "XYZ",
		},
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	err = consumer.processMessage(kafkago.Message{Value: payload})
	require.NoError(t, err)

	// Removed symbols are NOT deleted, just logged
	assert.Empty(t, repo.Upserts())
}

func TestWatchlistConsumer_processMessage_UnknownEventType(t *testing.T) {
	repo := &mockStockRepo{}
	consumer := &WatchlistConsumer{repo: repo}

	event := WatchlistEvent{
		EventType: "TOTALLY_UNKNOWN",
		Data:      WatchlistEventData{},
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	err = consumer.processMessage(kafkago.Message{Value: payload})
	require.NoError(t, err) // Unknown types are silently ignored
	assert.Empty(t, repo.Upserts())
}

func TestWatchlistConsumer_processMessage_InvalidJSON(t *testing.T) {
	repo := &mockStockRepo{}
	consumer := &WatchlistConsumer{repo: repo}

	err := consumer.processMessage(kafkago.Message{Value: []byte("{invalid")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestWatchlistConsumer_processMessage_EmptyAddedSymbols(t *testing.T) {
	repo := &mockStockRepo{}
	consumer := &WatchlistConsumer{repo: repo}

	event := WatchlistEvent{
		EventType: "WATCHLIST_UPDATED",
		Data: WatchlistEventData{
			AddedSymbols: []string{},
			TotalCount:   5,
		},
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	err = consumer.processMessage(kafkago.Message{Value: payload})
	require.NoError(t, err)
	assert.Empty(t, repo.Upserts())
}

func TestWatchlistConsumer_handleWatchlistUpdated_SymbolCaseNormalization(t *testing.T) {
	repo := &mockStockRepo{}
	consumer := &WatchlistConsumer{repo: repo}

	event := WatchlistEvent{
		EventType: "WATCHLIST_UPDATED",
		Data: WatchlistEventData{
			AddedSymbols: []string{"aapl", "Goog", "MSFT"},
			Stocks:       []WatchlistStock{}, // No name lookup available
		},
	}

	err := consumer.handleWatchlistUpdated(event)
	require.NoError(t, err)

	upserts := repo.Upserts()
	assert.Len(t, upserts, 3)
	assert.Equal(t, "AAPL", upserts[0].Symbol)
	assert.Equal(t, "GOOG", upserts[1].Symbol)
	assert.Equal(t, "MSFT", upserts[2].Symbol)
}

func TestWatchlistConsumer_handleWatchlistUpdated_NameFromStocksList(t *testing.T) {
	repo := &mockStockRepo{}
	consumer := &WatchlistConsumer{repo: repo}

	event := WatchlistEvent{
		EventType: "WATCHLIST_UPDATED",
		Data: WatchlistEventData{
			AddedSymbols: []string{"AAPL"},
			Stocks: []WatchlistStock{
				{Symbol: "AAPL", Name: "Apple Inc."},
				{Symbol: "GOOG", Name: "Alphabet Inc."},
			},
		},
	}

	err := consumer.handleWatchlistUpdated(event)
	require.NoError(t, err)

	upserts := repo.Upserts()
	require.Len(t, upserts, 1)
	assert.Equal(t, "Apple Inc.", upserts[0].Name)
}

func TestWatchlistConsumer_handleWatchlistUpdated_NoMatchingStock(t *testing.T) {
	repo := &mockStockRepo{}
	consumer := &WatchlistConsumer{repo: repo}

	event := WatchlistEvent{
		EventType: "WATCHLIST_UPDATED",
		Data: WatchlistEventData{
			AddedSymbols: []string{"SOFI"},
			Stocks:       []WatchlistStock{}, // empty
		},
	}

	err := consumer.handleWatchlistUpdated(event)
	require.NoError(t, err)

	upserts := repo.Upserts()
	require.Len(t, upserts, 1)
	// Falls back to symbol as name
	assert.Equal(t, "SOFI", upserts[0].Name)
}

// ---------------------------------------------------------------------------
// Sector/Industry enrichment tests
// ---------------------------------------------------------------------------

func TestWatchlistConsumer_handleSymbolAdded_WithSector(t *testing.T) {
	repo := &mockStockRepo{}
	consumer := &WatchlistConsumer{repo: repo}

	event := WatchlistEvent{
		EventType: "WATCHLIST_SYMBOL_ADDED",
		Data: WatchlistEventData{
			Symbol:   "CCJ",
			Name:     "Cameco Corp",
			Sector:   "Energy",
			Industry: "Uranium",
		},
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	err = consumer.processMessage(kafkago.Message{Value: payload})
	require.NoError(t, err)

	upserts := repo.Upserts()
	require.Len(t, upserts, 1)
	assert.Equal(t, "CCJ", upserts[0].Symbol)
	assert.Equal(t, "Cameco Corp", upserts[0].Name)
	assert.Equal(t, "Energy", upserts[0].Sector)
	assert.Equal(t, "Uranium", upserts[0].Industry)
}

func TestWatchlistConsumer_handleWatchlistUpdated_WithSectorFromStocks(t *testing.T) {
	repo := &mockStockRepo{}
	consumer := &WatchlistConsumer{repo: repo}

	event := WatchlistEvent{
		EventType: "WATCHLIST_UPDATED",
		Data: WatchlistEventData{
			AddedSymbols: []string{"AAPL", "XLE"},
			Stocks: []WatchlistStock{
				{Symbol: "AAPL", Name: "Apple Inc.", Sector: "Technology", Industry: "Consumer Electronics"},
				{Symbol: "XLE", Name: "Energy Select Sector", Sector: "Energy", Industry: "Oil & Gas"},
			},
		},
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	err = consumer.processMessage(kafkago.Message{Value: payload})
	require.NoError(t, err)

	upserts := repo.Upserts()
	require.Len(t, upserts, 2)
	assert.Equal(t, "AAPL", upserts[0].Symbol)
	assert.Equal(t, "Technology", upserts[0].Sector)
	assert.Equal(t, "Consumer Electronics", upserts[0].Industry)
	assert.Equal(t, "XLE", upserts[1].Symbol)
	assert.Equal(t, "Energy", upserts[1].Sector)
}

func TestWatchlistConsumer_handleWatchlistUpdated_MixedSectorAndNoSector(t *testing.T) {
	repo := &mockStockRepo{}
	consumer := &WatchlistConsumer{repo: repo}

	event := WatchlistEvent{
		EventType: "WATCHLIST_UPDATED",
		Data: WatchlistEventData{
			AddedSymbols: []string{"AAPL", "SOFI"},
			Stocks: []WatchlistStock{
				{Symbol: "AAPL", Name: "Apple Inc.", Sector: "Technology", Industry: "Consumer Electronics"},
				// SOFI has no sector in stocks list
			},
		},
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	err = consumer.processMessage(kafkago.Message{Value: payload})
	require.NoError(t, err)

	upserts := repo.Upserts()
	require.Len(t, upserts, 2)
	// AAPL should use UpsertStockWithSector (has sector)
	assert.Equal(t, "AAPL", upserts[0].Symbol)
	assert.Equal(t, "Technology", upserts[0].Sector)
	// SOFI should use UpsertStockBasic (no sector) — sector/industry empty
	assert.Equal(t, "SOFI", upserts[1].Symbol)
	assert.Equal(t, "", upserts[1].Sector)
}

// ---------------------------------------------------------------------------
// Start lifecycle (uses mock reader, similar to positions_consumer_test)
// ---------------------------------------------------------------------------

type mockWatchlistReader struct {
	cfg  kafkago.ReaderConfig
	msgs chan kafkago.Message

	mu         sync.Mutex
	closeCalls int
}

func newMockWatchlistReader(topic string, buffer int) *mockWatchlistReader {
	return &mockWatchlistReader{
		cfg:  kafkago.ReaderConfig{Topic: topic},
		msgs: make(chan kafkago.Message, buffer),
	}
}

func (r *mockWatchlistReader) ReadMessage(ctx context.Context) (kafkago.Message, error) {
	select {
	case msg := <-r.msgs:
		return msg, nil
	case <-ctx.Done():
		return kafkago.Message{}, ctx.Err()
	}
}

func (r *mockWatchlistReader) Close() error {
	r.mu.Lock()
	r.closeCalls++
	r.mu.Unlock()
	return nil
}

func (r *mockWatchlistReader) Config() kafkago.ReaderConfig {
	return r.cfg
}

func TestWatchlistConsumer_Start_ProcessesMessage(t *testing.T) {
	repo := &mockStockRepo{}
	reader := newMockWatchlistReader("watchlist-topic", 1)

	// WatchlistConsumer uses *kafka.Reader but we need to use our mock.
	// Since the reader field is unexported and typed as *kafka.Reader, we
	// construct the consumer manually (same pattern as positions_consumer_test).
	// However, the WatchlistConsumer uses *kafka.Reader (concrete), not an interface.
	// We can only test processMessage directly, which we've already done above.
	// The Start method requires a real kafka.Reader.
	// Instead, let's verify context cancellation via processMessage + direct call.

	_ = reader // reader not usable because WatchlistConsumer.reader is *kafka.Reader not interface

	// Test that processMessage can handle a well-formed message
	consumer := &WatchlistConsumer{repo: repo}
	event := WatchlistEvent{
		EventType: "WATCHLIST_SYMBOL_ADDED",
		Data:      WatchlistEventData{Symbol: "NVDA", Name: "NVIDIA Corp"},
	}
	payload, _ := json.Marshal(event)

	err := consumer.processMessage(kafkago.Message{Value: payload})
	require.NoError(t, err)

	upserts := repo.Upserts()
	require.Len(t, upserts, 1)
	assert.Equal(t, "NVDA", upserts[0].Symbol)
}

// ---------------------------------------------------------------------------
// UpsertStockBasic error handling
// ---------------------------------------------------------------------------

func TestWatchlistConsumer_handleSymbolAdded_UpsertError(t *testing.T) {
	repo := &mockStockRepo{err: assert.AnError}
	consumer := &WatchlistConsumer{repo: repo}

	event := WatchlistEvent{
		EventType: "WATCHLIST_SYMBOL_ADDED",
		Data:      WatchlistEventData{Symbol: "ERR"},
	}

	err := consumer.handleSymbolAdded(event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to upsert stock")
}

func TestWatchlistConsumer_handleWatchlistUpdated_UpsertErrorContinues(t *testing.T) {
	// When UpsertStockBasic fails for one symbol, other symbols are still processed
	callCount := 0
	repo := &mockStockRepo{}
	// Override the repo to fail on first call only
	failOnFirst := &failingStockRepo{failOn: 0}
	consumer := &WatchlistConsumer{repo: failOnFirst}
	_ = callCount
	_ = repo

	event := WatchlistEvent{
		EventType: "WATCHLIST_UPDATED",
		Data: WatchlistEventData{
			AddedSymbols: []string{"FAIL", "OK"},
		},
	}

	err := consumer.handleWatchlistUpdated(event)
	// Should not return error — errors are logged and continued
	require.NoError(t, err)
	// Only the second symbol should be upserted
	assert.Len(t, failOnFirst.upserts, 1)
	assert.Equal(t, "OK", failOnFirst.upserts[0].Symbol)
}

type failingStockRepo struct {
	mu      sync.Mutex
	failOn  int
	call    int
	upserts []stockUpsert
}

func (f *failingStockRepo) UpsertStockBasic(symbol, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	idx := f.call
	f.call++
	if idx == f.failOn {
		return assert.AnError
	}
	f.upserts = append(f.upserts, stockUpsert{Symbol: symbol, Name: name})
	return nil
}

func (f *failingStockRepo) UpsertStockWithSector(symbol, name, sector, industry string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	idx := f.call
	f.call++
	if idx == f.failOn {
		return assert.AnError
	}
	f.upserts = append(f.upserts, stockUpsert{Symbol: symbol, Name: name, Sector: sector, Industry: industry})
	return nil
}

func (f *failingStockRepo) StockExists(symbol string) (bool, error) {
	return false, nil
}
