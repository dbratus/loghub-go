// Copyright (C) 2014 Dmitry Bratus
//
// The use of this source code is governed by the license
// that can be found in the LICENSE file.

package loghub

import (
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	logOptions := ClientOptions{
		UseTls:             true,
		SkipCertValidation: true,
	}
	log := NewClient(":10001", &logOptions)

	for i := 0; i < 10; i++ {
		log.Write(1, "Test", "Test message.")
	}

	log.Close()

	<-time.After(time.Second)

	hubOptions := ClientOptions{
		UseTls:             true,
		SkipCertValidation: true,
		User:               "admin",
		Password:           "admin",
	}
	hub := NewClient(":10000", &hubOptions)
	defer hub.Close()

	sources := [...]string{"Test"}

	cnt := 0
	for _ = range hub.Read(time.Now().Add(-time.Minute), time.Now(), 0, 255, sources[:]) {
		cnt++
	}

	if cnt < 10 {
		t.Error("Entries has not been read.")
		t.FailNow()
	}

	hub.Truncate(time.Now(), "Test")

	<-time.After(time.Second)

	cnt = 0
	for _ = range hub.Read(time.Now().Add(-time.Minute), time.Now(), 0, 255, sources[:]) {
		cnt++
	}

	if cnt > 0 {
		t.Error("Entries has not been truncated.")
		t.FailNow()
	}

	cnt = 0
	for _ = range hub.Stat() {
		cnt++
	}

	if cnt == 0 {
		t.Error("Stat has not been returned.")
		t.FailNow()
	}
}
