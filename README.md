# LogHub client for Go

To learn about LogHub, visit [LogHub repository](https://github.com/dbratus/loghub).

## Getting the client

```
go get github.com/dbratus/loghub-go
```

## Using the client

Writing logs.

```Go
package main

import "github.com/dbratus/loghub-go"

func main() {
	log := loghub.NewClient(":10001", 1)
	defer log.Close()

	log.Write(1, "Example", "Example message.")
}
```

Reading logs.

```Go
package main

import (
	"github.com/dbratus/loghub-go"
	"time"
)

func main() {
	hub := loghub.NewClient(":10000", 1)
	defer hub.Close()

	sources := [...]string{"Test"}

	for _ = range hub.Read(time.Now().Add(-time.Minute), time.Now(), 0, 255, sources[:]) {
		// Consuming log entries.
	}
}
```

Truncating logs.

```Go
package main

import (
	"github.com/dbratus/loghub-go"
	"time"
)

func main() {
	hub := loghub.NewClient(":10000", 1)
	defer hub.Close()

	hub.Truncate(time.Now(), "Test")
}
```

Getting log stats.

```Go
package main

import (
	"github.com/dbratus/loghub-go"
	"time"
)

func main() {
	hub := loghub.NewClient(":10000", 1)
	defer hub.Close()

	for _ = range hub.Stat() {
		// Consuming stats.
	}
}
```
