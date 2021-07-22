//
// SPDX-License-Identifier: AGPL-3.0
//
/*******************************************************************************
 * Copyright (c) 2018 Genome Research Ltd.
 *
 * Author: Joshua C. Randall <jcrandall@alum.mit.edu>
 *
 * This file is part of Arvados.
 *
 * Arvados is free software: you can redistribute it and/or modify it under
 * the terms of the GNU Affero General Public License as published by the Free
 * Software Foundation; either version 3 of the License, or (at your option) any
 * later version.
 *
 * This program is distributed in the hope that it will be useful, but WITHOUT
 * ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
 * FOR A PARTICULAR PURPOSE. See the GNU Affero General Public License for more
 * details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 ******************************************************************************/

package main

import (
	"github.com/ceph/go-ceph/rados"
)

// radosRealImpl implements the radosImplementation interface by wrapping the real go-ceph/rados
type radosRealImpl struct{}

func (r *radosRealImpl) Version() (major, minor, patch int) {
	radosTracef("rados: Version()")
	major, minor, patch = rados.Version()
	radosTracef("rados: Version() complete, returning major=%d minor=%d patch=%d", major, minor, patch)
	return
}

func (r *radosRealImpl) NewConnWithClusterAndUser(clusterName string, userName string) (conn radosConn, err error) {
	radosTracef("rados: NewConnWithClusterAndUser clusterName=%s userName=%s", clusterName, userName)
	c, err := rados.NewConnWithClusterAndUser(clusterName, userName)
	conn = &radosRealConn{c}
	radosTracef("rados: NewConnWithClusterAndUser clusterName=%s userName=%s complete, returning conn=%+v err=%v", clusterName, userName, conn, err)
	return
}

// radosRealConn implements the radosConn interface by wraping *rados.Conn
type radosRealConn struct {
	*rados.Conn
}

// wrap OpenIOContext so we return radosIOContext instead of *rados.IOContext
func (conn *radosRealConn) OpenIOContext(pool string) (ioctx radosIOContext, err error) {
	i, err := conn.Conn.OpenIOContext(pool)
	ioctx = &radosRealIoctx{i}
	return
}

// radosRealIoctx implements the radosIOContext interface by wrapping *rados.IOContext
type radosRealIoctx struct {
	*rados.IOContext
}

// wrap Iter so we return radosIter instead of *rados.Iter
func (ioctx *radosRealIoctx) Iter() (iter radosIter, err error) {
	i, err := ioctx.IOContext.Iter()
	iter = &radosRealIter{i}
	return
}

// radosRealIter implements the radosIter interface by wrapping *rados.Iter
type radosRealIter struct {
	*rados.Iter
}
