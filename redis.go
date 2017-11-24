package main

import (
	rds "github.com/go-redis/redis"
	"log"
	"os"
)

var redis *rds.Client

const (
	KEY = "bitfinex"
	TICKER = "ticker"
)

func init() {
	client := rds.NewClient(&rds.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,  // use default DB
	})

	_, err := client.Ping().Result()

	if err != nil {
		log.Fatal(err.Error())
	}
	redis = client
}
