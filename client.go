// Copyright (C) 2014 Dmitry Bratus
//
// The use of this source code is governed by the license
// that can be found in the LICENSE file.

// This package provides a client for writing, reading and managing LogHub logs in Go.
package loghub

import (
	"github.com/dbratus/loghub/auth"
	"github.com/dbratus/loghub/lhproto"
	"time"
)

const (
	logBufFlushLen      = 100
	logBufFlushInterval = time.Millisecond * 100
)

// LogHub client.
type Client struct {
	client         lhproto.ProtocolHandler
	newEntriesChan chan *newLogEntry
	cred           lhproto.Credentials
	closeChan      chan chan bool
}

// LogHub log entry.
type LogEntry struct {
	Timestamp time.Time
	Severity  int
	Source    string
	Message   string
}

// Log information.
type LogInfo struct {
	// IP address and port of the log in the format 'ip:port'.
	Address string

	// The current size of the log in bytes.
	Size int64

	// The limit set on the log size.
	Limit int64
}

type newLogEntry struct {
	severity int
	source   string
	message  string
}

type ClientOptions struct {
	//How many connections may be maintained simultaneously.
	//If omitted, the default is 1.
	MaxConnections int

	//Whether to use TLS connection.
	UseTls bool

	//Whether to trust any certificate that the server returns.
	SkipCertValidation bool

	//User name.
	User string

	//User password.
	Password string
}

// Creates a new client connected to log or hub at specified address and port.
// Address may be specified in forms 'ip:port', 'hostname:port' or ':port' for
// local connection.
func NewClient(address string, options *ClientOptions) *Client {
	if options == nil {
		panic("Options must be specified.")
	}

	var maxConnections int

	if options.MaxConnections <= 0 {
		maxConnections = 1
	} else {
		maxConnections = options.MaxConnections
	}

	var cred lhproto.Credentials

	if options.User == "" {
		cred.User = auth.Anonymous
	} else {
		cred.User = options.User
		cred.Password = options.Password
	}

	cli := &Client{
		lhproto.NewClient(address, maxConnections, options.UseTls, options.SkipCertValidation),
		make(chan *newLogEntry),
		cred,
		make(chan chan bool),
	}

	go cli.writeLog()

	return cli
}

func (cli *Client) writeLog() {
	logBuf := make([]*newLogEntry, 0, logBufFlushLen)
	bufWrittenAt := time.Now()

	writeBuf := func(sync bool) {
		var backlogLen int

		if sync {
			backlogLen = 0
		} else {
			backlogLen = len(logBuf)
		}

		entries := make(chan *lhproto.IncomingLogEntryJSON, backlogLen)

		go cli.client.Write(&cli.cred, entries)

		for _, ent := range logBuf {
			entries <- &lhproto.IncomingLogEntryJSON{
				ent.severity,
				ent.source,
				ent.message,
			}
		}

		close(entries)
		logBuf = logBuf[:0]
		bufWrittenAt = time.Now()
	}

	for {
		select {
		case <-time.After(logBufFlushInterval):
			if time.Now().Sub(bufWrittenAt) >= logBufFlushInterval && len(logBuf) > 0 {
				writeBuf(false)
			}

		case ent, ok := <-cli.newEntriesChan:
			if ok {
				logBuf = append(logBuf, ent)

				if len(logBuf) == logBufFlushLen {
					writeBuf(false)
				}
			}

		case ack := <-cli.closeChan:
			for ent := range cli.newEntriesChan {
				logBuf = append(logBuf, ent)
			}

			if len(logBuf) > 0 {
				writeBuf(true)
			}

			ack <- true
			return
		}
	}
}

// Writes a log entry.
//
// Severity value must be within [0, 255] inclusively.
func (cli *Client) Write(severity int, source string, message string) {
	if severity < 0 || severity > 255 {
		panic("Invalid severity.")
	}

	cli.newEntriesChan <- &newLogEntry{severity, source, message}
}

// Reads log entries matching specified criteria returning them through the channel.
//
// If sources is nil or empty, the entries are returned regardless of the source.
func (cli *Client) Read(from, to time.Time, minSev, maxSev int, sources []string) chan *LogEntry {
	var queries chan *lhproto.LogQueryJSON
	entries := make(chan *lhproto.OutgoingLogEntryJSON)
	result := make(chan *LogEntry)

	if sources == nil || len(sources) == 0 {
		queries = make(chan *lhproto.LogQueryJSON, 1)
		queries <- &lhproto.LogQueryJSON{
			timeToTimestamp(from),
			timeToTimestamp(to),
			minSev,
			maxSev,
			"",
		}
	} else {
		queries = make(chan *lhproto.LogQueryJSON, len(sources))

		for _, src := range sources {
			queries <- &lhproto.LogQueryJSON{
				timeToTimestamp(from),
				timeToTimestamp(to),
				minSev,
				maxSev,
				src,
			}
		}
	}

	close(queries)

	go cli.client.Read(&cli.cred, queries, entries)

	go func() {
		for ent := range entries {
			result <- &LogEntry{
				timestampToLocalTime(ent.Ts),
				ent.Sev,
				ent.Src,
				ent.Msg,
			}
		}

		close(result)
	}()

	return result
}

// Truncates the log before the specified time.
// If source is an empty string, all sources are truncated.
func (cli *Client) Truncate(limit time.Time, source string) {
	cli.client.Truncate(&cli.cred, &lhproto.TruncateJSON{source, timeToTimestamp(limit)})
}

// Gets logs information.
func (cli *Client) Stat() chan *LogInfo {
	stats := make(chan *lhproto.StatJSON)
	result := make(chan *LogInfo)

	go cli.client.Stat(&cli.cred, stats)

	go func() {
		for stat := range stats {
			result <- &LogInfo{
				stat.Addr,
				stat.Sz,
				stat.Lim,
			}
		}

		close(result)
	}()

	return result
}

// Closes the client flushing the written entries.
func (cli *Client) Close() {
	close(cli.newEntriesChan)

	ack := make(chan bool)
	cli.closeChan <- ack
	<-ack

	cli.client.Close()
}

func timeToTimestamp(t time.Time) int64 {
	return t.UnixNano()
}

func timestampToLocalTime(t int64) time.Time {
	return time.Unix(0, t)
}
