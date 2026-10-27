package decorator

import (
	"strings"
	"time"
)

type MetricsClient interface {
	Inc(string, int)
}

func measure(client MetricsClient, namespace, action string, call func() error) (err error) {
	started := time.Now()
	key := namespace + "." + strings.ToLower(action)
	defer func() {
		client.Inc(key+".duration", int(time.Since(started)/time.Second))
		outcome := ".success"
		if err != nil {
			outcome = ".failure"
		}
		client.Inc(key+outcome, 1)
	}()
	return call()
}
