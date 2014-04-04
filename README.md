# LogHub client for Go

To learn about LogHub, visit [LogHub repository](https://github.com/dbratus/loghub).

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