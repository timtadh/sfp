package subgraph

/*
* This file is machine generated by fs2-generic. You can obtain
* fs2-generic from github.com/timtadh/fs2/fs2-generic . To install
*
*   $ go get github.com/timtadh/fs2
*   $ go install github.com/timtadh/fs2/fs2-generic
*
* The command used to generate the file was
*
*   $ fs2-generic \
*     --output=wrapper.go \
*     --package-name=subgraph \
*     mmlist \
*     --use-parameterized-serialization \
*     --item-type=*github.com/timtadh/goiso/SubGraph
*
* By including this code (and linking to github.com/timtadh/fs2) you
* accept the terms of the GPL version 3 unless another license has been
* obtained in writing from the author(s) of the package. The list of
* authors and full licensing information is located at
* https://github.com/timtadh/fs2/LICENSE
*
* This library is free software; you can redistribute it and/or modify
* it under the terms of the GNU General Public License as published by
* the Free Software Foundation; either version 3 of the License, or (at
* your option) any later version.
*
* This library is distributed in the hope that it will be useful, but
* WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
* General Public License for more details.
*
* You should have received a copy of the GNU General Public License
* along with this library; if not, write to the Free Software
* Foundation, Inc.,
*   51 Franklin Street, Fifth Floor,
*   Boston, MA  02110-1301
*   USA
*/

import (
	"sync"
)

import (
	"github.com/timtadh/fs2/fmap"
	"github.com/timtadh/fs2/mmlist"
)

import (
	"github.com/timtadh/goiso"
)


type List interface {
	Append(item *goiso.SubGraph) (i uint64, err error)
	Get(i uint64) (item *goiso.SubGraph, err error)
	Pop() (item *goiso.SubGraph, err error)
	Set(i uint64, item *goiso.SubGraph) (err error)
	Size() uint64
	Swap(i, j uint64) (err error)
	SwapDelete(i uint64) (item *goiso.SubGraph, err error)
	Close() error
	Delete() error
}

type MMList struct {
	bf *fmap.BlockFile
	list *mmlist.List
	mutex sync.Mutex
	serializeItem func(*goiso.SubGraph) []byte
	deserializeItem func([]byte) *goiso.SubGraph
}

func AnonList(
	serializeItem func(*goiso.SubGraph) []byte,
	deserializeItem func([]byte) *goiso.SubGraph,
) (*MMList, error) { 
	bf, err := fmap.Anonymous(fmap.BLOCKSIZE)
	if err != nil {
		return nil, err
	}
	return newMMList(bf, serializeItem, deserializeItem)
}

func NewList(
	path string,
	serializeItem func(*goiso.SubGraph) []byte,
	deserializeItem func([]byte) *goiso.SubGraph,
) (*MMList, error) { 
	bf, err := fmap.CreateBlockFile(path)
	if err != nil {
		return nil, err
	}
	return newMMList(bf, serializeItem, deserializeItem)
}

func OpenList(
	path string,
	serializeItem func(*goiso.SubGraph) []byte,
	deserializeItem func([]byte) *goiso.SubGraph,
) (*MMList, error) { 
	bf, err := fmap.OpenBlockFile(path)
	if err != nil {
		return nil, err
	}
	list, err := mmlist.Open(bf)
	if err != nil {
		return nil, err
	}
	b := &MMList{
		bf: bf,
		list: list,
		serializeItem: serializeItem,
		deserializeItem: deserializeItem,
	}
	return b, nil
}

func newMMList(
	bf *fmap.BlockFile,
	serializeItem func(*goiso.SubGraph) []byte,
	deserializeItem func([]byte) *goiso.SubGraph,
) (*MMList, error) { 
	list, err := mmlist.New(bf)
	if err != nil {
		return nil, err
	}
	b := &MMList{
		bf: bf,
		list: list,
		serializeItem: serializeItem,
		deserializeItem: deserializeItem,
	}
	return b, nil
}

func (m *MMList) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.bf.Close()
}

func (m *MMList) Delete() error {
	err := m.Close()
	if err != nil {
		return err
	}
	if m.bf.Path() != "" {
		return m.bf.Remove()
	}
	return nil
}

func (m *MMList) Size() uint64 {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.list.Size()
}

func (m *MMList) Append(item *goiso.SubGraph) (i uint64, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.list.Append(m.serializeItem(item))
}

func (m *MMList) Get(i uint64) (item *goiso.SubGraph, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	bytes, err := m.list.Get(i)
	if err != nil {
		return nil, err
	}
	return m.deserializeItem(bytes), nil
}

func (m *MMList) Pop() (item *goiso.SubGraph, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	bytes, err := m.list.Pop()
	if err != nil {
		return nil, err
	}
	return m.deserializeItem(bytes), nil
}

func (m *MMList) Set(i uint64, item *goiso.SubGraph) (err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.list.Set(i, m.serializeItem(item))
}

func (m *MMList) Swap(i, j uint64) (err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.list.Swap(i, j)
}

func (m *MMList) SwapDelete(i uint64) (item *goiso.SubGraph, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	bytes, err := m.list.SwapDelete(i)
	if err != nil {
		return nil, err
	}
	return m.deserializeItem(bytes), nil
}
