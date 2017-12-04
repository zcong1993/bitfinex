package main

import (
	"context"
	"fmt"
	"github.com/bitfinexcom/bitfinex-api-go/v2"
	"log"
	"sync"
	"time"
)

// Orderbook is struct of oaderbook
type Orderbook struct {
	mu *sync.Mutex
	// OK is if the data is available
	Ok bool
	// Symbol is market pair
	Symbol string
	// Bid is bid orderbook array now
	Bid SortableBook
	// Ask is ask orderbook array now
	Ask    SortableBook
	bidMap map[float64][]float64
	askMap map[float64][]float64
}

// SortableBook implement a sortable array struct
type SortableBook []SortableBookItem

// SortableBookItem is order struct for SortableBook
type SortableBookItem struct {
	Price  float64
	Amount float64
	Count  float64
}

// RunOrderbook start orderbook handler
func (bfx *Bfx) RunOrderbook() {
	c := bitfinex.NewClient()
	err := c.Websocket.Connect()
	if err != nil {
		log.Printf("Error connecting to web socket : %v", err)
		bfx.data.Ok = false
		bfx.orderbookDone <- struct{}{}
		return
	}
	c.Websocket.SetReadTimeout(time.Second * 10)
	c.Websocket.AttachEventHandler(func(ev interface{}) {
		log.Printf("EVENT: %#v", ev)
	})
	c.Websocket.AttachEventHandler(func(ev interface{}) {
		log.Printf("EVENT: %#v", ev)
	})
	for _, symbol := range bfx.Symbols {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		msg := &bitfinex.PublicSubscriptionRequest{
			Event:   "subscribe",
			Channel: bitfinex.ChanBook,
			Symbol:  bitfinex.TradingPrefix + symbol,
		}
		h := bfx.createOrderbookHandler(symbol)
		err = c.Websocket.Subscribe(ctx, msg, h)
		if err != nil {
			bfx.data.Ok = false
			bfx.orderbookDone <- struct{}{}
			return
		}
	}
	for {
		select {
		case <-c.Websocket.Done():
			log.Printf("channel closed: %s", c.Websocket.Err())
			bfx.data.Ok = false
			bfx.orderbookDone <- struct{}{}
			return
		}
	}
}

func (bfx *Bfx) createOrderbookHandler(symbol string) func(ev interface{}) {
	return func(ev interface{}) {
		bfx.orderbooks[symbol].mu.Lock()
		defer bfx.orderbooks[symbol].mu.Unlock()
		store := bfx.orderbooks[symbol]
		t, ok := ev.([][]float64)
		if ok {
			if len(t) > 1 {
				// full orderbook
				askMap, bidMap := splitOrderbook(t)
				store.askMap = askMap
				store.bidMap = bidMap
			} else if len(t) == 1 {
				// update
				ticker := t[0]
				p := ticker[0]
				c := ticker[1]
				a := ticker[2]
				if c == 0 {
					// delete
					if a == 1 {
						// delete from bid
						delete(store.bidMap, p)
					} else {
						// delete from ask
						delete(store.askMap, p)
					}
				} else if c > 0 {
					// update
					if a > 0 {
						// update bids
						store.bidMap[p] = ticker
					} else {
						// update asks
						store.askMap[p] = ticker
					}
				}
			}
			// TODO update getter update redis
			bfx.updateOrderbook(symbol)
		} else {
			//log.Printf("PUBLIC MSG HEARTBEAT %s: %#v", symbol, ev)
		}
	}
}

func (bfx *Bfx) updateOrderbook(symbol string) {
	store := bfx.orderbooks[symbol]
	bid := SortableBook{}
	ask := SortableBook{}
	for _, b := range store.bidMap {
		item := SortableBookItem{Price: b[0], Count: b[1], Amount: b[2]}
		bid = append(bid, item)
	}
	for _, a := range store.askMap {
		item := SortableBookItem{Price: a[0], Count: a[1], Amount: a[2]}
		ask = append(ask, item)
	}
	store.Bid = bid
	store.Ask = ask
	store.Ok = true
	fmt.Printf("%+v\n\n", bfx.orderbooks[symbol])
}

func splitOrderbook(book [][]float64) (map[float64][]float64, map[float64][]float64) {
	ask := make(map[float64][]float64)
	bid := make(map[float64][]float64)
	for _, b := range book {
		if len(b) != 3 {
			continue
		}
		amount := b[2]
		price := b[0]
		if amount > 0 {
			bid[price] = b
		} else {
			ask[price] = b
		}
	}
	return ask, bid
}
