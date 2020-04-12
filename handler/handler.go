package handler

import (
	"github.com/Sirupsen/logrus"
	"github.com/kulikov/go-sbus"
	"github.com/servehub/utils/gabs"
)

type Handler interface {
	Run(bus *sbus.Sbus, conf *gabs.Container, log *logrus.Entry) error
}

var HandlerRegestry = &handlerRegestry{}

type handlerRegestry struct {
	handlers map[string]Handler
}

func (r *handlerRegestry) Add(name string, handler Handler) {
	if r.handlers == nil {
		r.handlers = make(map[string]Handler)
	}

	if _, ok := r.handlers[name]; ok {
		logrus.Fatalf("Handler '%s' duplicate name", name)
	}

	r.handlers[name] = handler
}

func (r *handlerRegestry) Get(name string) Handler {
	h, ok := r.handlers[name]
	if !ok {
		logrus.Fatalf("Handler '%s' doesn't exist!", name)
	}
	return h
}

func (r *handlerRegestry) Has(name string) bool {
	_, ok := r.handlers[name]
	return ok
}
