/*
Copyright 2013 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lru

import (
	"fmt"
	"testing"
	"time"
)

type simpleStruct struct {
	int
	string
}

type complexStruct struct {
	int
	simpleStruct
}

var getTests = []struct {
	name       string
	keyToAdd   interface{}
	keyToGet   interface{}
	expectedOk bool
}{
	{"string_hit", "myKey", "myKey", true},
	{"string_miss", "myKey", "nonsense", false},
	{"simple_struct_hit", simpleStruct{1, "two"}, simpleStruct{1, "two"}, true},
	{"simple_struct_miss", simpleStruct{1, "two"}, simpleStruct{0, "noway"}, false},
	{"complex_struct_hit", complexStruct{1, simpleStruct{2, "three"}},
		complexStruct{1, simpleStruct{2, "three"}}, true},
}

func TestGet(t *testing.T) {
	for _, tt := range getTests {
		lru := New(0)
		lru.Add(tt.keyToAdd, 1234, time.Time{})
		val, ok := lru.Get(tt.keyToGet)
		if ok != tt.expectedOk {
			t.Fatalf("%s: cache hit = %v; want %v", tt.name, ok, !ok)
		} else if ok && val != 1234 {
			t.Fatalf("%s expected get to return 1234 but got %v", tt.name, val)
		}
	}
}

func TestRemove(t *testing.T) {
	lru := New(0)
	lru.Add("myKey", 1234, time.Time{})
	if val, ok := lru.Get("myKey"); !ok {
		t.Fatal("TestRemove returned no match")
	} else if val != 1234 {
		t.Fatalf("TestRemove failed.  Expected %d, got %v", 1234, val)
	}

	lru.Remove("myKey")
	if _, ok := lru.Get("myKey"); ok {
		t.Fatal("TestRemove returned a removed entry")
	}
}

func TestEvict(t *testing.T) {
	evictedKeys := make([]Key, 0)
	onEvictedFun := func(key Key, value interface{}) {
		evictedKeys = append(evictedKeys, key)
	}

	lru := New(20)
	lru.OnEvicted = onEvictedFun
	for i := 0; i < 22; i++ {
		lru.Add(fmt.Sprintf("myKey%d", i), 1234, time.Time{})
	}

	if len(evictedKeys) != 2 {
		t.Fatalf("got %d evicted keys; want 2", len(evictedKeys))
	}
	if evictedKeys[0] != Key("myKey0") {
		t.Fatalf("got %v in first evicted key; want %s", evictedKeys[0], "myKey0")
	}
	if evictedKeys[1] != Key("myKey1") {
		t.Fatalf("got %v in second evicted key; want %s", evictedKeys[1], "myKey1")
	}
}

func TestExpire(t *testing.T) {
	var tests = []struct {
		name       string
		key        interface{}
		expectedOk bool
		expire     time.Duration
		wait       time.Duration
	}{
		{"not-expired", "myKey", true, time.Second * 1, time.Duration(0)},
		{"expired", "expiredKey", false, time.Millisecond * 100, time.Millisecond * 150},
	}

	for _, tt := range tests {
		lru := New(0)
		lru.Add(tt.key, 1234, time.Now().Add(tt.expire))
		time.Sleep(tt.wait)
		val, ok := lru.Get(tt.key)
		if ok != tt.expectedOk {
			t.Fatalf("%s: cache hit = %v; want %v", tt.name, ok, !ok)
		} else if ok && val != 1234 {
			t.Fatalf("%s expected get to return 1234 but got %v", tt.name, val)
		}
	}
}

func TestReplacement(t *testing.T) {
	replacedKeys := make([]Key, 0)
	onReplacementFunc := func(key Key, value interface{}, expire time.Time) {
		replacedKeys = append(replacedKeys, key)
	}

	lru := New(2)
	lru.OnReplace = onReplacementFunc
	for i := 0; i < 3; i++ {
		lru.Add(fmt.Sprintf("myKey%d", i%2), 1234, time.Time{})
	}

	if len(replacedKeys) != 1 {
		t.Fatalf("got %d evicted keys; want 1", len(replacedKeys))
	}
	if replacedKeys[0] != Key("myKey0") {
		t.Fatalf("got %v in first evicted key; want %s", replacedKeys[0], "myKey0")
	}
}
