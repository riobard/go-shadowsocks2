package speeddial

import (
	"log"
	"net"
	"sync/atomic"
	"time"
)

var epoch = time.Now()

const weight = 4
const penalty = 2 * time.Second

const _Debug = false

func logf(format string, v ...interface{}) {
	if _Debug {
		log.Printf(format, v...)
	}
}

type Dial func() (net.Conn, error)

type target struct {
	dial     Dial
	last     int64 // last dial since epoch
	latency  int64 // exponetially smoothed
	inflight int32 // number of inflight dial
}

func (t *target) Dial() (net.Conn, error) {
	atomic.AddInt32(&t.inflight, 1)
	defer atomic.AddInt32(&t.inflight, -1)
	t0 := time.Now()
	if old, new := atomic.LoadInt64(&t.last), t0.Sub(epoch).Nanoseconds(); old < new {
		atomic.CompareAndSwapInt64(&t.last, old, new)
	}
	c, err := t.dial()
	latency := time.Since(t0).Nanoseconds()
	if err != nil {
		latency = int64(penalty)
	}
	old := atomic.LoadInt64(&t.latency)
	if old > 0 {
		latency = (weight*old + latency) / (weight + 1) // exponentially weighted moving average
	}
	atomic.CompareAndSwapInt64(&t.latency, old, latency)
	return c, err
}

type Dialer struct {
	targets  []target
	Cooldown time.Duration // default 10 seconds
}

func New(ds ...Dial) *Dialer {
	tgts := make([]target, len(ds))
	for i := range ds {
		tgts[i].dial = ds[i]
	}
	return &Dialer{targets: tgts, Cooldown: 10 * time.Second}
}

func (d *Dialer) Dial() (net.Conn, error) {
	min := int64(1<<63 - 1)
	var best int
	for i := range d.targets {
		if l := atomic.LoadInt64(&d.targets[i].latency); 0 < l && l < min {
			best, min = i, l
		}
	}

	logf("Best #%d [%dms]", best, min/1e6)

	for i := range d.targets {
		if i == best {
			continue
		}
		tgt := &d.targets[i]
		if t := atomic.LoadInt64(&tgt.last); t > 0 && time.Since(epoch.Add(time.Duration(t))) < d.Cooldown {
			logf("Cool down #%d", i)
			continue
		}
		if n := atomic.LoadInt32(&tgt.inflight); n > 0 {
			logf("Inflight #%d [%d]", i, n)
			continue
		}
		go func(i int) {
			c, err := tgt.Dial()
			if err == nil {
				c.Close()
			}
			logf("Latency #%d [%dms]", i, tgt.latency/1e6)
		}(i)
	}

	return d.targets[best].Dial()
}
