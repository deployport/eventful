package main

import (
	"log"
	"runtime"
	"time"
)

func main() {
	ch := make(chan struct{})
	run := func() {
		go func() {
			defer log.Printf("ending A")
			log.Printf("starting A")
			ch <- struct{}{}
		}()
		go func() {
			defer log.Printf("ending B")
			log.Printf("starting B")
			ch <- struct{}{}
		}()
	}
	run()
	log.Printf("running gc")
	runtime.GC()
	log.Printf("ran gc")
	// close(ch)
	<-ch
	time.Sleep(time.Second)
}
