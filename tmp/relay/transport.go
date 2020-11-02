package main

type Transport interface {
	Send([]byte) error
	Recv() <-chan []byte
}
