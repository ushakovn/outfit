package worker

import (
  "context"
  "sync"

  log "github.com/sirupsen/logrus"
)

const DefaultCount = 5

type Call func(ctx context.Context) error

type Pool struct {
  count   uint8
  ch      chan Call
  done    chan struct{}
  stopped bool
}

func NewPool(ctx context.Context, count uint8) Pool {
  pool := Pool{
    count: count,
    ch:    make(chan Call),
    done:  make(chan struct{}),
  }
  pool.start(ctx)

  return pool
}

func (p *Pool) start(ctx context.Context) {
  var wg sync.WaitGroup

  wg.Add(int(p.count))

  for index := 0; index < int(p.count); index++ {
    go func() {
      defer wg.Done()

      for {
        select {
        case <-ctx.Done():
          log.Warn("worker.pool: context cancelled: worker stopped")
          return

        case call, ok := <-p.ch:
          if !ok {
            return
          }
          if err := call(ctx); err != nil {
            log.Errorf("worker.pool: worker call failed: %v", err)
          }
        }
      }
    }()
  }

  go func() {
    wg.Wait()

    p.done <- struct{}{}
  }()

}

func (p *Pool) Push(call Call) {
  p.ch <- call
}

func (p *Pool) StopWait() {
  if p.stopped {
    return
  }
  close(p.ch)

  <-p.done

  p.stopped = true
}
