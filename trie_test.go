// Copyright 2012 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package trie

import (
	"bytes"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

var tcAA = []string{
	"aardvark",
	"abro",
	"abrocome",
	"abrogable",
	"abrogate",
	"abrogation",
	"abrogative",
	"abrogator",
	"abronah",
	"abroniaaaaa",
	"abroniaaaab",
	"abroniaaa",
}

func mktc() (Trie, map[string]bool) {
	m := make(map[string]bool, len(tcAA))
	for _, s := range tcAA {
		m[s] = true
	}

	var tr Trie
	for s := range m { // getting them from m gives a randomized order
		tr.Put(s, s)
	}
	return tr, m
}

func TestThatItWorks(t *testing.T) {
	tr, m := mktc()
	//tr.dump(1)
	//t.Error(tr.shape())
	// We can retrieve what we put in
	for s := range m {
		if v, ok := tr.Get(s).(string); !ok || v != s {
			if ok {
				t.Error("tr[", s, "] == ", v, ", expecting ", s)
			} else {
				t.Error("tr[", s, "] == nil, expecting ", s)
			}
		}
	}

	// we don't retrieve any prefixes (except explicitly inserted ones)
	for s := range m {
		for i := 0; i < len(s)-1; i++ {
			if m[s[:i]] {
				continue
			}
			if v := tr.Get(s[:i]); v != nil {
				t.Error("tr[", s[:i], "] == ", v, ", expecting nil")
			}
		}
	}

	// ForEach reproduces them all in sorted order
	prev := ""
	tr.ForEach(func(s string, val interface{}) bool {
		if v, ok := val.(string); !ok || v != s {
			if ok {
				t.Error("tr[", s, "] == ", v, ", expecting ", s)
			} else {
				t.Error("tr[", s, "] == nil, expecting ", s)
			}
		}

		if !m[s] {
			t.Error("tr[", s, "] == ", val, ", but should not exist")
		}

		if prev >= s {
			t.Errorf("out of order element: %+v after %+v", s, prev)
		}
		prev = s

		delete(m, s)
		return true
	})

	// ForEach exhausts
	if len(m) > 0 {
		t.Error("Unretrieved: ", m)
	}
}

func TestForEachPfx(t *testing.T) {
	tr, m := mktc()
	//tr.dump(1)

	const testPfx = "abr"
	m2 := make(map[string]bool)
	for s := range m {
		if len(s) > len(testPfx) && s[:len(testPfx)] == testPfx {
			m2[s] = true
		}
	}

	// ForEachPfx reproduces them all in sorted order
	prev := ""
	tr.ForEachPfx(testPfx, func(s string, val interface{}) bool {
		if v, ok := val.(string); !ok || v != s {
			if ok {
				t.Error("tr[", s, "] == ", v, ", expecting ", s)
			} else {
				t.Error("tr[", s, "] == nil, expecting ", s)
			}
		}

		if !m[s] {
			t.Error("tr[", s, "] == ", val, ", but should not exist")
		}

		if prev >= s {
			t.Errorf("out of order element: %+v after %+v", s, prev)
		}
		prev = s

		delete(m2, s)
		return true
	})

	// ForEachPfx exhausts
	if len(m2) > 0 {
		t.Error("Unretrieved: ", m)
	}
}

func TestFindPfx(t *testing.T) {
	var tr Trie
	tr.Put("fooaaaa", 1)
	tr.Put("foocdef", 2)

	for _, tc := range []struct {
		key, pfx string
		val      interface{}
	}{
		{"", "", nil},
		{"a", "", nil},
		{"f", "", nil},
		{"foo", "", nil},
		{"fooaaaa", "fooaaaa", 1},
		{"fooaaaabcd", "fooaaaa", 1},
		{"foob", "", nil},
		{"foocd", "", nil},
		{"foocdef", "foocdef", 2},
		{"foocdefgh", "foocdef", 2},
		{"g", "", nil},
	} {
		if k, v := tr.FindPfx(tc.key); k != tc.pfx || v != tc.val {
			t.Errorf("FindPfx(%q) = %q, %v,  expecting %q, %v", tc.key, k, v, tc.pfx, tc.val)
		}
	}
}

func TestFindAllPfx(t *testing.T) {
	var tr Trie
	tr.Put("aaaa", 1)
	tr.Put("aabb", 2)
	tr.Put("aa", 3)

	for _, tc := range []struct {
		key string
		kvs []KV
	}{
		{"aa", []KV{{"aa", 3}}},
		{"aabb", []KV{{"aabb", 2}, {"aa", 3}}},
		{"aabbcc", []KV{{"aabb", 2}, {"aa", 3}}},
	} {
		if kvs := tr.FindAllPfx(tc.key); !reflect.DeepEqual(kvs, tc.kvs) {
			t.Errorf("FindAllPfx(%q), expected %+v, got %+v", tc.key, tc.kvs, kvs)
		}
	}
}

// Benchmarks to compare inserting random strings into a map or a trie and retrieving them in sorted order
// generate 10000 strings from a limited alphabet (8 characters) to get a fair probability of shared prefixes.
const alphabet = 8

var tc []string

func init() {
	var b bytes.Buffer
	m := make(map[string]bool)
	for len(m) < 100000 {
		b.Reset()
		for l := rand.Intn(4) + 1; l > 0; l-- {
			ch := byte(65 + rand.Intn(alphabet))
			for r := rand.Intn(4) + 1; r > 0; r-- {
				b.WriteByte(ch)
			}
		}
		m[b.String()] = true
	}
	for s := range m {
		tc = append(tc, s)
	}
}

// just insertion, no retrieval
func nativeMap(size int) {
	m := make(map[string]string, size)
	for _, s := range tc[:size] {
		m[s] = s
	}
}

// insertion and get all in sorted order
func nativeMapAndSort(size int) {
	m := make(map[string]string, size)
	for _, s := range tc[:size] {
		m[s] = s
	}
	sl := make([]string, len(m))
	for k := range m {
		sl = append(sl, k)
	}
	sort.Sort(sort.StringSlice(sl))
}

// just insertion, no retrieval
func withTrie(size int) {
	var tr Trie
	for _, s := range tc[:size] {
		tr.Put(s, s)
	}
}

// insertion and get all in sorted order
func withTrieAndAll(size int) {
	var tr Trie
	for i, s := range tc {
		if i > size {
			break
		}
		tr.Put(s, s)
	}

	tr.ForEach(func(key string, val interface{}) bool { return true })
}

func BenchmarkNativeMap10(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nativeMap(10)
	}
}
func BenchmarkNativeMap100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nativeMap(100)
	}
}
func BenchmarkNativeMap1000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nativeMap(1000)
	}
}
func BenchmarkNativeMap10000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nativeMap(10000)
	}
}
func BenchmarkNativeMap100000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nativeMap(100000)
	}
}

func BenchmarkNativeMapAndSort10(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nativeMapAndSort(10)
	}
}
func BenchmarkNativeMapAndSort100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nativeMapAndSort(100)
	}
}
func BenchmarkNativeMapAndSort1000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nativeMapAndSort(1000)
	}
}
func BenchmarkNativeMapAndSort10000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nativeMapAndSort(10000)
	}
}
func BenchmarkNativeMapAndSort100000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nativeMapAndSort(100000)
	}
}

func BenchmarkWithTrie10(b *testing.B) {
	for i := 0; i < b.N; i++ {
		withTrie(10)
	}
}
func BenchmarkWithTrie100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		withTrie(100)
	}
}
func BenchmarkWithTrie1000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		withTrie(1000)
	}
}
func BenchmarkWithTrie10000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		withTrie(10000)
	}
}
func BenchmarkWithTrie100000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		withTrie(100000)
	}
}

func BenchmarkWithTrieAndAll10(b *testing.B) {
	for i := 0; i < b.N; i++ {
		withTrieAndAll(10)
	}
}
func BenchmarkWithTrieAndAll100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		withTrieAndAll(100)
	}
}
func BenchmarkWithTrieAndAll1000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		withTrieAndAll(1000)
	}
}
func BenchmarkWithTrieAndAll10000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		withTrieAndAll(10000)
	}
}
func BenchmarkWithTrieAndAll100000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		withTrieAndAll(100000)
	}
}

func forEach(size int, b *testing.B) {
	b.StopTimer()
	var tr Trie
	for _, s := range tc[:size] {
		tr.Put(s, s)
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		i := 0
		tr.ForEach(func(key string, val interface{}) bool {
			i++
			if key != val.(string) {
				b.Error(key, " != ", val.(string))
				return false
			}
			return true
		})

		if i != size {
			b.Error("aah", i)
		}
	}
}

func BenchmarkForEach1(b *testing.B)     { forEach(1, b) }
func BenchmarkForEach10(b *testing.B)    { forEach(10, b) }
func BenchmarkForEach100(b *testing.B)   { forEach(100, b) }
func BenchmarkForEach1000(b *testing.B)  { forEach(1000, b) }
func BenchmarkForEach10000(b *testing.B) { forEach(10000, b) }

func byteEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func forEachB(size int, b *testing.B) {
	b.StopTimer()
	var tr Trie
	for _, s := range tc[:size] {
		tr.Put(s, []byte(s))
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		i := 0
		tr.ForEachB(func(key []byte, val interface{}) bool {
			i++
			if !byteEqual(key, val.([]byte)) {
				b.Error(key, " != ", val.([]byte))
				return false
			}
			return true
		})

		if i != size {
			b.Error("aah", i)
		}
	}
}

func BenchmarkForEachB1(b *testing.B)     { forEachB(1, b) }
func BenchmarkForEachB10(b *testing.B)    { forEachB(10, b) }
func BenchmarkForEachB100(b *testing.B)   { forEachB(100, b) }
func BenchmarkForEachB1000(b *testing.B)  { forEachB(1000, b) }
func BenchmarkForEachB10000(b *testing.B) { forEachB(10000, b) }
