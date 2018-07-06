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
	"fmt"
	"os"
	"time"

	"github.com/ceph/go-ceph/rados"
)

// radosMockImpl implements the radosImplemetation interface for testing purposes
type radosMockImpl struct {
	b *radosStubBackend
}

func (r *radosMockImpl) Version() (major int, minor int, patch int) {
	radosTracef("radosmock: Version() calling rados.Version()")
	// might as well pass this along to the actual librados client
	return rados.Version()
}

func (r *radosMockImpl) NewConnWithClusterAndUser(clusterName string, userName string) (conn radosConn, err error) {
	radosTracef("radosmock: NewConnWithClusterAndUser clusterName=%s userName=%s", clusterName, userName)
	conn = &radosMockConn{
		radosMockImpl: r,
		cluster:       clusterName,
		user:          userName,
	}
	radosTracef("radosmock: NewConnWithClusterAndUser clusterName=%s userName=%s complete, returning conn=%+v err=%v", clusterName, userName, conn, err)
	return
}

type radosMockConn struct {
	*radosMockImpl
	cluster   string
	user      string
	connected bool
}

func (conn *radosMockConn) SetConfigOption(option, value string) (err error) {
	radosTracef("radosmock: conn.SetConfigOption option=%s value=%s")
	conn.b.Lock()
	defer conn.b.Unlock()

	conn.b.config[option] = value
	radosTracef("radosmock: conn.SetConfigOption option=%s value=%s complete, returning err=%v", err)
	return
}

func (conn *radosMockConn) Connect() (err error) {
	radosTracef("radosmock: conn.Connect()")
	conn.b.Lock()
	defer conn.b.Unlock()

	conn.connected = true
	conn.b.fsid = RadosMockFSID
	for _, pool := range RadosMockPools {
		conn.b.pools[pool] = newRadosStubPool()
	}
	radosTracef("radosmock: conn.Connect() complete, returning err=%v", err)
	return
}

func (conn *radosMockConn) GetFSID() (fsid string, err error) {
	radosTracef("radosmock: conn.GetFSID()")
	conn.b.Lock()
	defer conn.b.Unlock()

	fsid = conn.b.fsid
	if !conn.connected {
		err = fmt.Errorf("radosmock: GetFSID called before Connect")
	}
	radosTracef("radosmock: conn.GetFSID() complete, returning fsid=%s err=%v", fsid, err)
	return
}

func (conn *radosMockConn) GetClusterStats() (stat rados.ClusterStat, err error) {
	radosTracef("radosmock: conn.GetClusterStats()")
	conn.b.Lock()
	defer conn.b.Unlock()

	if !conn.connected {
		panic("radosmock: GetClusterStats called before Connect")
	}
	stat.Kb = conn.b.totalSize
	for _, pool := range conn.b.pools {
		for _, namespace := range pool.namespaces {
			for _, obj := range namespace.objects {
				size := len(obj.data)
				stat.Kb_used += uint64(size)
				stat.Num_objects++
			}
		}
	}
	stat.Kb_avail = stat.Kb - stat.Kb_used
	radosTracef("radosmock: conn.GetClusterStats() complete, returning stat=%+v err=%v", stat, err)
	return
}

func (conn *radosMockConn) ListPools() (names []string, err error) {
	radosTracef("radosmock: conn.ListPools()")
	conn.b.Lock()
	defer conn.b.Unlock()

	names = make([]string, len(conn.b.pools))
	i := 0
	for k := range conn.b.pools {
		names[i] = k
		i++
	}
	if !conn.connected {
		err = fmt.Errorf("radosmock: ListPools called before Connect")
	}
	radosTracef("radosmock: conn.ListPools() complete, returning names=%v err=%v", names, err)
	return
}

func (conn *radosMockConn) OpenIOContext(pool string) (ioctx radosIOContext, err error) {
	radosTracef("radosmock: conn.OpenIOContext pool=%s", pool)
	ioctx = &radosMockIoctx{
		radosMockConn: conn,
		pool:          pool,
	}
	ioctx.SetNamespace("")

	radosTracef("radosmock: conn.OpenIOContext pool=%s complete, returning ioctx=%+v err=%v", pool, ioctx, err)
	return
}

func (conn *radosMockConn) Shutdown() {
	radosTracef("radosmock: conn.Shutdown()")
}

type radosMockIoctx struct {
	*radosMockConn
	pool      string
	namespace string
	objects   map[string]*radosStubObj
}

func (ioctx *radosMockIoctx) Delete(oid string) (err error) {
	radosTracef("radosmock: ioctx.Delete oid=%s", oid)
	ioctx.b.Lock()
	defer ioctx.b.Unlock()

	_, ok := ioctx.objects[oid]
	if !ok {
		err = rados.RadosErrorNotFound
		return
	}
	delete(ioctx.objects, oid)
	radosTracef("radosmock: ioctx.Delete oid=%s complete, returning err=%v", oid, err)
	return
}

func (ioctx *radosMockIoctx) GetPoolStats() (stat rados.PoolStat, err error) {
	radosTracef("radosmock: ioctx.GetPoolStats()")
	ioctx.b.Lock()
	defer ioctx.b.Unlock()

	pool := ioctx.b.pools[ioctx.pool]
	for _, namespace := range pool.namespaces {
		for _, obj := range namespace.objects {
			size := len(obj.data)
			stat.Num_bytes += uint64(size)
			stat.Num_objects++
		}
	}
	stat.Num_kb = stat.Num_bytes / 1024
	stat.Num_object_clones = stat.Num_objects * ioctx.b.numReplicas

	radosTracef("radosmock: ioctx.GetPoolStats() complete, returning stat=%+v err=%v", stat, err)
	return
}

func (ioctx *radosMockIoctx) GetXattr(oid string, name string, data []byte) (n int, err error) {
	radosTracef("radosmock: ioctx.GetXattr oid=%s name=%s len(data)=%d", oid, name, len(data))
	ioctx.b.Lock()
	defer ioctx.b.Unlock()

	obj, ok := ioctx.objects[oid]
	if !ok {
		err = rados.RadosErrorNotFound
		radosTracef("radosmock: ioctx.GetXattr oid=%s name=%s len(data)=%d object not found in ioctx.objects, returning n=%d err=%v", oid, name, len(data), n, err)
		return
	}
	xv, ok := obj.xattrs[name]
	if !ok {
		err = rados.RadosErrorNotFound
		radosTracef("radosmock: ioctx.GetXattr oid=%s name=%s len(data)=%d object found but name not in xattrs, returning n=%d err=%v", oid, name, len(data), n, err)
		return
	}
	n = copy(data, xv)
	radosTracef("radosmock: ioctx.GetXattr oid=%s name=%s len(data)=%d populated data='%s', returning n=%d err=%v", oid, name, len(data), data, n, err)
	return
}

func (ioctx *radosMockIoctx) Iter() (iter radosIter, err error) {
	radosTracef("radosmock: ioctx.Iter()")
	ioctx.b.Lock()
	defer ioctx.b.Unlock()

	oids := make([]string, len(ioctx.objects))
	i := 0
	for oid := range ioctx.objects {
		oids[i] = oid
		i++
	}
	iter = &radosMockIter{
		radosMockIoctx: ioctx,
		oids:           oids,
		current:        -1,
	}
	radosTracef("radosmock: ioctx.Iter() complete, returning iter=%+v err=%v", iter, err)
	return
}

func (ioctx *radosMockIoctx) LockExclusive(oid, name, cookie, desc string, duration time.Duration, flags *byte) (res int, err error) {
	return ioctx.lock(oid, name, cookie, true)
}

func (ioctx *radosMockIoctx) lock(oid, name, cookie string, exclusive bool) (res int, err error) {
	radosTracef("radosmock: ioctx.lock oid=%s name=%s cookie=%s exclusive=%v", oid, name, cookie, exclusive)
	ioctx.b.Lock()
	defer ioctx.b.Unlock()

	_, ok := ioctx.objects[oid]
	if !ok {
		// locking a nonexistant object creates an empty object
		ioctx.objects[oid] = newRadosStubObj([]byte{})
	}
	obj, ok := ioctx.objects[oid]
	if !ok {
		err = fmt.Errorf("radosmock: failed to create nonexistant object for lock")
		radosTracef("radosmock: ioctx.lock oid=%s name=%s cookie=%s exclusive=%v failed to create object, returning err=%v", oid, name, cookie, exclusive, err)
		return
	}

	existingCookie, exclusiveLockHeld := obj.exclusiveLocks[name]
	if exclusiveLockHeld {
		if exclusive && existingCookie == cookie {
			res = RadosLockExist
		} else {
			res = RadosLockBusy
		}
		radosTracef("radosmock: ioctx.lock oid=%s name=%s cookie=%s exclusive=%v exclusive lock already held, returning err=%v", oid, name, cookie, exclusive, err)
		return
	}

	existingCookieMap, sharedLockHeld := obj.sharedLocks[name]
	if sharedLockHeld {
		if exclusive {
			// want an exclusive lock but shared locks exist
			res = RadosLockBusy
		} else {
			// want a shared lock
			_, sharedLockExist := existingCookieMap[cookie]
			if sharedLockExist {
				res = RadosLockExist
			} else {
				// want a shared lock and some exist but not ours, add our cookie to the map
				existingCookieMap[cookie] = true
				res = RadosLockLocked
			}
		}
		radosTracef("radosmock: ioctx.lock oid=%s name=%s cookie=%s exclusive=%v shared lock already held, returning err=%v", oid, name, cookie, exclusive, err)
		return
	}

	// there is no existing lock by this name on this object, take the lock
	if exclusive {
		obj.exclusiveLocks[name] = cookie
	} else {
		obj.sharedLocks[name] = make(map[string]bool)
		obj.sharedLocks[name][cookie] = true
	}
	res = RadosLockLocked
	radosTracef("radosmock: ioctx.lock oid=%s name=%s cookie=%s exclusive=%v no existing lock, lock obtained, returning err=%v", oid, name, cookie, exclusive, err)
	return
}

func (ioctx *radosMockIoctx) LockShared(oid, name, cookie, tag, desc string, duration time.Duration, flags *byte) (res int, err error) {
	return ioctx.lock(oid, name, cookie, false)
}

func (ioctx *radosMockIoctx) Read(oid string, data []byte, offset uint64) (n int, err error) {
	radosTracef("radosmock: ioctx.Read oid=%s len(data)=%d offset=%d", oid, len(data), offset)
	ioctx.b.Lock()
	defer ioctx.b.Unlock()

	obj, ok := ioctx.objects[oid]
	if !ok {
		err = os.ErrNotExist
		return
	}
	n = copy(data, obj.data[offset:])

	// pause here to facilitate race tests
	radosTracef("radosmock: ioctx.Read oid=%s len(data)=%d offset=%d calling unlockAndRace()", oid, len(data), offset)
	ioctx.b.unlockAndRace()

	radosTracef("radosmock: ioctx.Read oid=%s len(data)=%d offset=%d complete, returning n=%d err=%v", oid, len(data), offset, n, err)
	return
}

func (ioctx *radosMockIoctx) SetNamespace(namespace string) {
	radosTracef("radosmock: ioctx.SetNamespace namespace=%s", namespace)
	ioctx.b.Lock()
	defer ioctx.b.Unlock()

	ioctx.namespace = namespace
	_, ok := ioctx.b.pools[ioctx.pool].namespaces[ioctx.namespace]
	if !ok {
		ioctx.b.pools[ioctx.pool].namespaces[ioctx.namespace] = newRadosStubNamespace()
	}
	ns, _ := ioctx.b.pools[ioctx.pool].namespaces[ioctx.namespace]
	ioctx.objects = ns.objects
	radosTracef("radosmock: ioctx.SetNamespace namespace=%s complete, returning", namespace)
	return
}

func (ioctx *radosMockIoctx) SetXattr(oid string, name string, data []byte) (err error) {
	radosTracef("radosmock: ioctx.SetXattr oid=%s name=%s len(data)=%d data='%s'", oid, name, len(data), data)
	ioctx.b.Lock()
	defer ioctx.b.Unlock()

	obj, ok := ioctx.objects[oid]
	if !ok {
		err = rados.RadosErrorNotFound
		return
	}
	d := make([]byte, len(data))
	copy(d, data)
	obj.xattrs[name] = d
	radosTracef("radosmock: ioctx.SetXattr oid=%s name=%s len(data)=%d data='%s' complete, returning err=%v", oid, name, len(data), data, err)
	return
}

func (ioctx *radosMockIoctx) Stat(oid string) (stat rados.ObjectStat, err error) {
	radosTracef("radosmock: ioctx.Stat oid=%s", oid)
	ioctx.b.Lock()
	defer ioctx.b.Unlock()

	obj, ok := ioctx.objects[oid]
	if !ok {
		err = os.ErrNotExist
		radosTracef("radosmock: ioctx.Stat oid=%s object does not exist, returning stat=%+v err=%v", oid, stat, err)
		return
	}
	stat.Size = uint64(len(obj.data))
	// don't bother implementing stat.ModTime as we do not use it

	radosTracef("radosmock: ioctx.Stat oid=%s complete, returning stat=%+v err=%v", oid, stat, err)
	return
}

func (ioctx *radosMockIoctx) Truncate(oid string, size uint64) (err error) {
	radosTracef("radosmock: ioctx.Truncate oid=%s size=%d", oid, size)
	ioctx.b.Lock()
	defer ioctx.b.Unlock()

	obj, ok := ioctx.objects[oid]
	if !ok {
		err = os.ErrNotExist
		radosTracef("radosmock: ioctx.Truncate oid=%s size=%d object not found, returning err=%v", oid, size, err)
		return
	}

	if uint64(len(obj.data)) < size {
		// existing data is smaller than truncation size, pad it with zeros
		d := make([]byte, size)
		copy(d, obj.data[:])
		obj.data = d
		radosTracef("radosmock: ioctx.Truncate oid=%s size=%d enlarged object from %d bytes by padding with zeros", oid, size, len(obj.data))
	}

	if uint64(len(obj.data)) > size {
		// existing data is larger than truncation size
		d := make([]byte, size)
		copy(d, obj.data[:size])
		obj.data = d
		radosTracef("radosmock: ioctx.Truncate oid=%s size=%d shrunk object from %d bytes", oid, size, len(obj.data))
	}

	radosTracef("radosmock: ioctx.Truncate oid=%s size=%d complete, returning err=%v", oid, size, err)
	return
}

func (ioctx *radosMockIoctx) Unlock(oid, name, cookie string) (res int, err error) {
	radosTracef("radosmock: ioctx.Unlock oid=%s name=%s cookie=%s", oid, name, cookie)
	ioctx.b.Lock()
	defer ioctx.b.Unlock()

	obj, ok := ioctx.objects[oid]
	if !ok {
		res = RadosLockNotFound
		return
	}

	existingCookie, exclusiveLockHeld := obj.exclusiveLocks[name]
	if exclusiveLockHeld {
		if existingCookie == cookie {
			// this is our lock, delete it
			delete(obj.exclusiveLocks, name)
			res = RadosLockUnlocked
		} else {
			res = RadosLockNotFound
		}
		radosTracef("radosmock: ioctx.Unlock oid=%s name=%s cookie=%s, returning res=%v err=%v", oid, name, cookie, res, err)
		return
	}

	existingCookieMap, sharedLockHeld := obj.sharedLocks[name]
	if sharedLockHeld {
		_, sharedLockExist := existingCookieMap[cookie]
		if sharedLockExist {
			// this is our cookie, delete it from the cookie map
			delete(existingCookieMap, cookie)
			if len(existingCookieMap) == 0 {
				// this was the last shared cookie, delete the sharedLocks entry as well
				delete(obj.sharedLocks, name)
			}
			res = RadosLockUnlocked
		} else {
			res = RadosLockNotFound
		}
		radosTracef("radosmock: ioctx.Unlock oid=%s name=%s cookie=%s found existing shared lock(s) but none matched the provided cookie, returning res=%v err=%v", oid, name, cookie, res, err)
		return
	}

	res = RadosLockNotFound
	radosTracef("radosmock: ioctx.Unlock oid=%s name=%s cookie=%s no lock found to unlock, returning res=%v err=%v", oid, name, cookie, res, err)
	return
}

func (ioctx *radosMockIoctx) WriteFull(oid string, data []byte) (err error) {
	radosTracef("radosmock: ioctx.WriteFull oid=%s len(data)=%d", oid, len(data))
	ioctx.b.Lock()
	defer ioctx.b.Unlock()

	obj, ok := ioctx.objects[oid]
	if !ok {
		ioctx.objects[oid] = newRadosStubObj([]byte{})
	}

	// pause here to facilitate race tests
	radosTracef("radosmock: ioctx.WriteFull oid=%s len(data)=%d calling unlockAndRace()", oid, len(data))
	ioctx.b.unlockAndRace()

	obj.data = make([]byte, len(data))
	n := copy(obj.data, data)
	if n != len(data) {
		err = fmt.Errorf("radosmock: WriteFull for oid=%s was expected to copy %d bytes but only copied %d", oid, len(data), n)
	}
	radosTracef("radosmock: ioctx.WriteFull oid=%s len(data)=%d complete, returning err=%v", oid, len(data), err)
	return
}

type radosMockIter struct {
	*radosMockIoctx
	oids    []string
	current int
}

func (iter *radosMockIter) Next() (next bool) {
	radosTracef("radosmock: iter.Next() iter.current=%d len(iter.oids)=%d", iter.current, len(iter.oids))
	iter.current++
	next = true
	if iter.current >= len(iter.oids) {
		next = false
	}
	radosTracef("radosmock: iter.Next() iter.current=%d len(iter.oids)=%d, returning next=%v", iter.current, len(iter.oids), next)
	return
}

func (iter *radosMockIter) Value() (value string) {
	radosTracef("radosmock: iter.Value() iter.current=%d len(iter.oids)=%d", iter.current, len(iter.oids))
	if iter.current >= 0 && iter.current < len(iter.oids) {
		value = iter.oids[iter.current]
	}
	radosTracef("radosmock: iter.Value() iter.current=%d len(iter.oids)=%d, returning value=%s", iter.current, len(iter.oids), value)
	return
}

func (iter *radosMockIter) Close() {
	radosTracef("radosmock: iter.Close() iter.current=%d len(iter.oids)=%d", iter.current, len(iter.oids))
	return
}
