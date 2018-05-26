package main

import (
	"net/http"
	"sync"
)

type Pool struct {
	mutex   sync.Mutex
	maxSize int
	minSize int
  currentSize int
	head    *PoolNode
	tail    *PoolNode
}

type PoolNode struct {
	client *http.Client
	next   *PoolNode
}

func (p *Pool) init(minSize int, maxSize int) {
	p.mutex = sync.Mutex{}
  p.minSize = sync.
  for i:=0; i<minSize; i++ {
    client := http.Client{}
    node := PoolNode{}
  }
}

func (p *Pool) acquire() http.Client {
}

func (p *Pool) release() {

}
