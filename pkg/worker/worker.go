package worker

import (
  "context"
  "sync"

  log "github.com/sirupsen/logrus"
)

const DefaultCount = 5

type Call func(ctx context.Context) error

type Pool struct {
  count uint8
  ch    chan Call
}

func NewPool(ctx context.Context, count uint8) Pool {
  pool := Pool{
    count: count,
    ch:    make(chan Call),
  }
  go pool.start(ctx)

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

        case call := <-p.ch:
          if err := call(ctx); err != nil {
            log.Errorf("worker.pool: worker call failed: %v", err)
          }
        }
      }
    }()
  }

  wg.Wait()
}

func (p *Pool) Push(call Call) {
  p.ch <- call
}
