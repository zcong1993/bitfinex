package main

import (
	"context"
	"encoding/json"
	"github.com/bitfinexcom/bitfinex-api-go/v2"
	"log"
	"sync"
	"time"
)

// Bfx is bitfinex wrapper client
type Bfx struct {
	// Symbols are the ticker pairs you want to subscribe
	Symbols []string
	data    Data
}

// Data is tickers data for redis
type Data struct {
	mu sync.Mutex
	// OK is if the data is available
	Ok bool
	// Tickers is the ticker data
	Tickers map[string][][]float64
	// Last is the last price map
	Last map[string]float64
}

// NewBfx create a Bfx instance
func NewBfx(Symbols []string) *Bfx {
	data := Data{
		Ok:      false,
		Tickers: map[string][][]float64{},
		Last:    map[string]float64{},
	}
	b := &Bfx{Symbols: Symbols, data: data}
	go b.run()
	return b
}

func (bfx *Bfx) run() {
	c := bitfinex.NewClient()
	err := c.Websocket.Connect()
	if err != nil {
		log.Fatal("Error connecting to web socket : ", err)
	}
	c.Websocket.SetReadTimeout(time.Second * 10)
	c.Websocket.AttachEventHandler(func(ev interface{}) {
		log.Printf("EVENT: %#v", ev)
	})
	c.Websocket.AttachEventHandler(func(ev interface{}) {
		log.Printf("EVENT: %#v", ev)
	})
	for _, symbol := range bfx.Symbols {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		msg := &bitfinex.PublicSubscriptionRequest{
			Event:   "subscribe",
			Channel: bitfinex.ChanTicker,
			Symbol:  bitfinex.TradingPrefix + symbol,
		}
		h := bfx.createTickerHandler(symbol)
		err = c.Websocket.Subscribe(ctx, msg, h)
		if err != nil {
			log.Fatal(err)
		}
	}
	for {
		select {
		case <-c.Websocket.Done():
			log.Printf("channel closed: %s", c.Websocket.Err())
			bfx.data.Ok = false
			bfx.run()
			return
		}
	}
}

func (bfx *Bfx) createTickerHandler(symbol string) func(ev interface{}) {
	return func(ev interface{}) {
		t, ok := ev.([][]float64)
		if ok {
			last := t[0][6]
			bfx.data.mu.Lock()
			bfx.data.Tickers[symbol] = t
			bfx.data.Last[symbol] = last
			bfx.data.Ok = true
			d, _ := json.Marshal(bfx.data)
			redis.HSet(KEY, TICKER, string(d))
			bfx.data.mu.Unlock()
			log.Printf("PUBLIC MSG %s: %#v", symbol, last)
		} else {
			//log.Printf("PUBLIC MSG HEARTBEAT %s: %#v", symbol, ev)
		}
	}
}

func main() {
	NewBfx(Symbols)
	select {}
}
