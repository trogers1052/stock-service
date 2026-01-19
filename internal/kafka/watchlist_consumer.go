package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// StockRepository defines the interface for stock database operations
type StockRepository interface {
	UpsertStockBasic(symbol, name string) error
	StockExists(symbol string) (bool, error)
}

// WatchlistEvent represents a watchlist event from Kafka
type WatchlistEvent struct {
	EventType string             `json:"event_type"`
	Source    string             `json:"source"`
	Timestamp string             `json:"timestamp"`
	Data      WatchlistEventData `json:"data"`
}

// WatchlistEventData holds the data for different watchlist event types
type WatchlistEventData struct {
	// For WATCHLIST_UPDATED events
	AddedSymbols   []string         `json:"added_symbols,omitempty"`
	RemovedSymbols []string         `json:"removed_symbols,omitempty"`
	AllSymbols     []string         `json:"all_symbols,omitempty"`
	TotalCount     int              `json:"total_count,omitempty"`
	Stocks         []WatchlistStock `json:"stocks,omitempty"`

	// For WATCHLIST_SYMBOL_ADDED/REMOVED events
	Symbol string `json:"symbol,omitempty"`
	Name   string `json:"name,omitempty"`
}

// WatchlistStock represents stock details in the event
type WatchlistStock struct {
	Symbol        string `json:"symbol"`
	Name          string `json:"name"`
	InstrumentURL string `json:"instrument_url"`
	AddedAt       string `json:"added_at"`
}

// WatchlistConsumer handles consuming watchlist events from Kafka
type WatchlistConsumer struct {
	reader *kafka.Reader
	repo   StockRepository
}

// NewWatchlistConsumer creates a new Kafka consumer for watchlist events
func NewWatchlistConsumer(brokers []string, topic, groupID string, repo StockRepository) *WatchlistConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID + "-watchlist",
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		MaxWait:        1 * time.Second,
		StartOffset:    kafka.FirstOffset,
		CommitInterval: time.Second,
	})

	return &WatchlistConsumer{
		reader: reader,
		repo:   repo,
	}
}

// Start begins consuming messages from Kafka
func (c *WatchlistConsumer) Start(ctx context.Context) error {
	log.Printf("Starting watchlist consumer for topic: %s", c.reader.Config().Topic)

	for {
		select {
		case <-ctx.Done():
			log.Println("Watchlist consumer shutting down...")
			return c.reader.Close()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return nil // Context cancelled, normal shutdown
				}
				log.Printf("Error reading watchlist message: %v", err)
				continue
			}

			if err := c.processMessage(msg); err != nil {
				log.Printf("Error processing watchlist message: %v", err)
				// Continue processing other messages
			}
		}
	}
}

// processMessage handles a single Kafka message
func (c *WatchlistConsumer) processMessage(msg kafka.Message) error {
	log.Printf("Received watchlist message from partition %d offset %d: key=%s",
		msg.Partition, msg.Offset, string(msg.Key))

	var event WatchlistEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal watchlist event: %w", err)
	}

	log.Printf("Processing watchlist event: %s", event.EventType)

	switch event.EventType {
	case "WATCHLIST_UPDATED":
		return c.handleWatchlistUpdated(event)

	case "WATCHLIST_SYMBOL_ADDED":
		return c.handleSymbolAdded(event)

	case "WATCHLIST_SYMBOL_REMOVED":
		// For now, we don't delete stocks when removed from watchlist
		// We just log it - the stock data may still be useful
		log.Printf("Symbol removed from watchlist: %s (keeping in database)",
			event.Data.Symbol)
		return nil

	default:
		log.Printf("Ignoring unknown watchlist event type: %s", event.EventType)
		return nil
	}
}

// handleWatchlistUpdated processes a full watchlist update event
func (c *WatchlistConsumer) handleWatchlistUpdated(event WatchlistEvent) error {
	log.Printf("Processing watchlist update: %d added, %d removed, %d total",
		len(event.Data.AddedSymbols),
		len(event.Data.RemovedSymbols),
		event.Data.TotalCount)

	// Process added symbols
	for _, symbol := range event.Data.AddedSymbols {
		symbol = strings.ToUpper(symbol)
		name := symbol

		// Find name from stocks list
		for _, stock := range event.Data.Stocks {
			if strings.ToUpper(stock.Symbol) == symbol {
				name = stock.Name
				break
			}
		}

		if err := c.repo.UpsertStockBasic(symbol, name); err != nil {
			log.Printf("Error upserting stock %s: %v", symbol, err)
			continue
		}
		log.Printf("Added/updated stock: %s (%s)", symbol, name)
	}

	return nil
}

// handleSymbolAdded processes a single symbol added event
func (c *WatchlistConsumer) handleSymbolAdded(event WatchlistEvent) error {
	symbol := strings.ToUpper(event.Data.Symbol)
	name := event.Data.Name
	if name == "" {
		name = symbol
	}

	if err := c.repo.UpsertStockBasic(symbol, name); err != nil {
		return fmt.Errorf("failed to upsert stock %s: %w", symbol, err)
	}

	log.Printf("Added/updated stock from watchlist: %s (%s)", symbol, name)
	return nil
}

// Close closes the Kafka consumer
func (c *WatchlistConsumer) Close() error {
	return c.reader.Close()
}
