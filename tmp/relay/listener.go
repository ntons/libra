package main

type Listener interface {
	Accept() Transport
}
