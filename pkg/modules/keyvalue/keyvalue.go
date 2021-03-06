// Copyright 2020 Vikunja and contriubtors. All rights reserved.
//
// This file is part of Vikunja.
//
// Vikunja is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Vikunja is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Vikunja.  If not, see <https://www.gnu.org/licenses/>.

package keyvalue

import (
	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/modules/keyvalue/memory"
	"code.vikunja.io/api/pkg/modules/keyvalue/redis"
)

// Storage defines an interface for saving key-value pairs
type Storage interface {
	Put(key string, value interface{}) (err error)
	Get(key string) (value interface{}, err error)
	Del(key string) (err error)
	IncrBy(key string, update int64) (err error)
	DecrBy(key string, update int64) (err error)
}

var store Storage

// InitStorage initializes the configured storage backend
func InitStorage() {
	switch config.KeyvalueType.GetString() {
	case "redis":
		store = redis.NewStorage()
	case "memory":
		fallthrough
	default:
		store = memory.NewStorage()
	}
}

// Put puts a value in the storage backend
func Put(key string, value interface{}) error {
	return store.Put(key, value)
}

// Get returns a value from a storage backend
func Get(key string) (value interface{}, err error) {
	return store.Get(key)
}

// Del removes a save value from a storage backend
func Del(key string) (err error) {
	return store.Del(key)
}

// IncrBy increases a value at key by the amount in update
func IncrBy(key string, update int64) (err error) {
	return store.IncrBy(key, update)
}

// DecrBy increases a value at key by the amount in update
func DecrBy(key string, update int64) (err error) {
	return store.DecrBy(key, update)
}
