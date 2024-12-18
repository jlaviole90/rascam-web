package main

import (
	"context"
	"fmt"
	"image/jpeg"
	"net/http"
	"os"
	"sync"
	"time"

    "nhooyr.io/websocket"
	camera "rascam-web/internal"
)

type server struct {
	mux              http.ServeMux
	subscribers      map[*subscriber]struct{}
	subscribersMutex sync.Mutex
	messageBuffer    int
}

type subscriber struct {
	msgs chan []byte
}

func NewServer() *server {
	s := &server{
		messageBuffer: 10,
		subscribers:   make(map[*subscriber]struct{}),
	}

	s.mux.Handle("/", http.FileServer(http.Dir("./htmx")))
	s.mux.HandleFunc("/ws", s.subscribeHandler)
	return s
}

func (s *server) subscribeHandler(w http.ResponseWriter, r *http.Request) {
	if err := s.subscribe(r.Context(), w, r); err != nil {
		fmt.Println(err)
		return
	}
}

func (s *server) addSubscriber(sub *subscriber) {
	s.subscribersMutex.Lock()
	s.subscribers[sub] = struct{}{}
	s.subscribersMutex.Unlock()
	fmt.Println("Added subscriber", sub)
}

func (s *server) subscribe(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var c *websocket.Conn
	subscriber := &subscriber{
		msgs: make(chan []byte, s.messageBuffer),
	}
	s.addSubscriber(subscriber)

	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		return err
	}
	defer c.Close(websocket.StatusAbnormalClosure, "goodbye")

	ctx = c.CloseRead(ctx)
	for {
		select {
		case msg := <-subscriber.msgs:
			ctx, cancel := context.WithTimeout(ctx, time.Second*5)
			defer cancel()
			err := c.Write(ctx, websocket.MessageText, msg)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *server) publishMsg(msg []byte) {
	s.subscribersMutex.Lock()
	defer s.subscribersMutex.Unlock()

	for sub := range s.subscribers {
		sub.msgs <- msg
	}
}

func main() {
	fmt.Println("Starting server on :8080")
	s := NewServer()

	go func(s *server) {
		for {
            if err := camera.Capture(); err != nil {
                fmt.Println(err)
                continue
            }

			var opts jpeg.Options
			opts.Quality = 1

			timeStamp := time.Now().Format(time.RFC3339)
			msg := []byte(`
            <img hx-swap-oob="innerHTML:#update-frame" src="../data/frame.jpg"> 
            <div hx=swap-oob="innerHTML:#update-timestamp">` + timeStamp + `</div>
            `)

			s.publishMsg(msg)
			time.Sleep(3 * time.Millisecond)
		}
	}(s)

	err := http.ListenAndServe(":8080", &s.mux)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
