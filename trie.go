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

// Package trie implements a byte trie with edge compression.
package trie

import (
	"bytes"
	"fmt"
)

// A Trie maintains a sorted collection of values keyed on a string.
// Insertion is O(len(key)). Unlike Go's built-in map there is no
// distinction between a nil and a non-existent value.
// The zero value for Trie is an empty trie ready to use.
type Trie struct {
	suffix   string
	value    interface{}
	children []Trie
	base     byte
}

// Find the largest i such that a[:i] == b[:i]
func commonPrefix(a, b string) int {
	l := len(a)
	if l > len(b) {
		l = len(b)
	}
	for i := 0; i < l; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return l
}

// Put inserts or replaces a value in the trie.  To remove a value
// insert nil.
func (t *Trie) Put(key string, value interface{}) {
	if t.children == nil && t.value == nil { // empty node
		t.suffix = key
		t.value = value
		return
	}

	s := commonPrefix(t.suffix, key)

	if s < len(t.suffix) {
		// split on s: turn t into a node with suffix[:s]
		// and move the contents to child[suffix[s]-t.base] with suffix[s+1:]
		// we save the extra alloc on the common case that we'd insert a subtrie
		// on key[s] immediately below by making children large enough
		newbase := t.suffix[s]
		newlen := 1
		if s < len(key) {
			for key[s] < newbase || int(key[s]) >= int(newbase)+newlen {
				newlen *= 2
				newbase &= ^byte(newlen - 1)
			}
		}
		newch := make([]Trie, newlen)
		newch[t.suffix[s]-newbase] = Trie{
			suffix:   t.suffix[s+1:],
			value:    t.value,
			children: t.children,
			base:     t.base,
		}

		t.suffix = t.suffix[:s]
		t.value = nil
		t.children = newch
		t.base = newbase
	}

	if s == len(key) {
		t.value = value
		return
	}

	if len(t.children) == 0 {
		t.children = make([]Trie, 1)
		t.base = key[s]
	} else {
		newbase := t.base
		newlen := len(t.children)
		for key[s] < newbase || int(key[s]) >= int(newbase)+newlen {
			newlen *= 2
			newbase &= ^byte(newlen - 1)
		}
		if newlen != len(t.children) {
			newch := make([]Trie, newlen)
			copy(newch[t.base-newbase:], t.children)
			t.children = newch
			t.base = newbase
		}
	}

	t.children[key[s]-t.base].Put(key[s+1:], value)

	return
}

// Get retrieves an element from the trie if it exists, or nil if it does not.
func (t *Trie) Get(key string) interface{} {
	s := commonPrefix(t.suffix, key)

	if s < len(t.suffix) {
		return nil
	}

	if s == len(key) {
		return t.value
	}

	if key[s] < t.base || int(key[s]) >= int(t.base)+len(t.children) {
		return nil
	}

	return t.children[key[s]-t.base].Get(key[s+1:])
}

// FindPfx finds the longest prefix of key in the trie that has a non-nil value.
func (t *Trie) FindPfx(key string) (pfx string, val interface{}) {
	s := commonPrefix(t.suffix, key)

	if s < len(t.suffix) {
		return "", nil
	}

	if s == len(key) { // and s == len(t.suffix) , so key = t.suffix
		if t.value != nil {
			return t.suffix, t.value
		}
		return "", nil
	}
	// there's a bit of key left over.  if it is out of range, we're the longest prefix

	if key[s] < t.base || int(key[s]) >= int(t.base)+len(t.children) {
		if t.value != nil {
			return t.suffix, t.value
		}
		return "", nil
	}

	p, v := t.children[key[s]-t.base].FindPfx(key[s+1:])
	if v != nil {
		return key[:s+1] + p, v
	}

	if t.value != nil {
		return t.suffix, t.value
	}
	return "", nil
}

// A KV is a key, value pair, as returned by FindAllPfx
type KV struct {
	K string
	V interface{}
}

// FindAllPfx returns all prefixes of key in the trie and their values
// in order of longest to shortest!
func (t *Trie) FindAllPfx(key string) []KV { return t.findAllPfx(key, 0) }

func (t *Trie) findAllPfx(key string, ofs int) []KV {
	s := commonPrefix(t.suffix, key[ofs:])

	if s < len(t.suffix) {
		return nil
	}

	if ofs+s == len(key) { // and s == len(t.suffix) , so key[ofs:] = t.suffix
		if t.value != nil {
			return []KV{{key[:ofs+s], t.value}}
		}
		return nil
	}
	// there's a bit of key left over.  if it is out of range, we're the longest prefix

	if key[s] < t.base || int(key[s]) >= int(t.base)+len(t.children) {
		if t.value != nil {
			return []KV{{key[:ofs+s], t.value}}
		}
		return nil
	}

	kv := t.children[key[s]-t.base].findAllPfx(key, ofs+s+1)
	if t.value != nil {
		kv = append(kv, KV{key[:ofs+s], t.value})
	}
	return kv
}

// subtrie retrieves the part of t that has key as a prefix.
// and the part of its suffix that should be tacked on to key
func (t *Trie) subtrie(key string) (*Trie, int) {
	s := commonPrefix(t.suffix, key)

	if s == len(key) {
		return t, s
	}

	if s < len(t.suffix) {
		return nil, 0
	}

	// s == len(suffix) but s < len(key): there's a bit of key left over

	if key[s] < t.base || int(key[s]) >= int(t.base)+len(t.children) {
		return nil, 0
	}

	return t.children[key[s]-t.base].subtrie(key[s+1:])
}

func (t *Trie) forEach(f func([]byte, interface{}) bool, buf *bytes.Buffer) bool {
	if t.value == nil && t.children == nil {
		return true
	}

	pfx := buf.Len()
	buf.WriteString(t.suffix)

	if t.value != nil && !f(buf.Bytes(), t.value) { // this is the alloc (probably)
		return false
	}

	if t.children != nil {
		l := buf.Len()
		buf.WriteByte(t.base)
		for _, v := range t.children {
			if !v.forEach(f, buf) {
				return false
			}
			buf.Bytes()[l]++
		}
	}

	buf.Truncate(pfx)

	return true
}

// ForEach will apply the function f to each key, value pair in the
// Trie in sorted (depth-first pre-)order.  if f returns false, the
// iteration will stop.
func (t *Trie) ForEach(f func(string, interface{}) bool) {
	var buf bytes.Buffer
	t.forEach(func(b []byte, v interface{}) bool { return f(string(b), v) }, &buf)
}

// ForEachB will apply the function f to each key, value pair in the
// Trie in sorted (depth-first pre-)order.  if f returns false, the
// iteration will stop.
func (t *Trie) ForEachB(f func([]byte, interface{}) bool) {
	var buf bytes.Buffer
	t.forEach(f, &buf)
}

// ForEachPfx applies ForEach to the sub-trie that starts with the given prefix.
func (t *Trie) ForEachPfx(pfx string, f func(string, interface{}) bool) {
	t, sfx := t.subtrie(pfx)
	if t == nil {
		return
	}
	var buf bytes.Buffer
	buf.Grow(len(pfx))
	buf.WriteString(pfx[:len(pfx)-sfx])
	t.forEach(func(b []byte, v interface{}) bool { return f(string(b), v) }, &buf)
}

// String returns a multiline string representation of the trie
// in the form
//    trie[
//       key1: value1
//       key2: value2
//       ....
//    ]
func (t *Trie) String() string {
	var buf bytes.Buffer
	buf.WriteString("trie{\n")
	t.ForEach(func(key string, val interface{}) bool {
		fmt.Fprintf(&buf, "\t%s:%v\n", key, val)
		return true
	})
	buf.WriteString("}")
	return buf.String()
}

// debug
const spaces = "                                                                                "

func (t *Trie) dump(level int) {
	if level > len(spaces) {
		level = len(spaces)
	}
	fmt.Printf("%s: %v\n", t.suffix, t.value)
	if t.children != nil {
		fmt.Printf("%s<%d>\n", spaces[:4*level], len(t.children))
	}
	c := t.base
	for _, ch := range t.children {
		if ch.value != nil || ch.children != nil {
			if c >= 32 && c < 128 {
				fmt.Printf("%s['%c']", spaces[:4*level], c)
			} else {
				fmt.Printf("%s[%d]", spaces[:4*level], c)
			}
			ch.dump(level + 1)
		}
		c++
	}
}

func (t *Trie) shape() (ln, sz int) {
	if t.value != nil {
		ln++
	}
	sz++
	for _, c := range t.children {
		l, s := c.shape()
		ln += l
		sz += s
	}
	return
}
