// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-2020 Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package models

import "code.vikunja.io/web"

// CanCreate checks if a user can create a new bucket
func (b *Bucket) CanCreate(a web.Auth) (bool, error) {
	l := &List{ID: b.ListID}
	return l.CanWrite(a)
}

// CanUpdate checks if a user can update an existing bucket
func (b *Bucket) CanUpdate(a web.Auth) (bool, error) {
	return b.canDoBucket(a)
}

// CanDelete checks if a user can delete an existing bucket
func (b *Bucket) CanDelete(a web.Auth) (bool, error) {
	return b.canDoBucket(a)
}

// canDoBucket checks if the bucket exists and if the user has the right to act on it
func (b *Bucket) canDoBucket(a web.Auth) (bool, error) {
	bb, err := getBucketByID(x.NewSession(), b.ID)
	if err != nil {
		return false, err
	}
	l := &List{ID: bb.ListID}
	return l.CanWrite(a)
}
