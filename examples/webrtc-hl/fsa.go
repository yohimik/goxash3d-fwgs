package main

import (
	"errors"
	"sync"
)

type FixedArray[T any] struct {
	data     []T
	inUse    []bool
	capacity byte
	lock     sync.RWMutex
	freeList []byte
}

// NewFixedArray initializes the fixed-size array
func NewFixedArray[T any](size byte) *FixedArray[T] {
	free := make([]byte, size)
	var i byte = 0
	for ; i < size; i++ {
		free[i] = size - 1 - i
	}
	return &FixedArray[T]{
		data:     make([]T, size),
		inUse:    make([]bool, size),
		capacity: size,
		freeList: free,
	}
}

// Add inserts a value and returns its index (O(1))
func (fa *FixedArray[T]) Add(item T) (byte, error) {
	fa.lock.Lock()
	defer fa.lock.Unlock()

	if len(fa.freeList) == 0 {
		return 0, errors.New("array is full")
	}
	idx := fa.freeList[len(fa.freeList)-1]
	fa.freeList = fa.freeList[:len(fa.freeList)-1]

	fa.data[idx] = item
	fa.inUse[idx] = true
	return idx, nil
}

// Get retrieves a value by index (O(1))
func (fa *FixedArray[T]) Get(index byte) (T, error) {
	fa.lock.RLock()
	defer fa.lock.RUnlock()

	var zero T
	if index < 0 || index >= fa.capacity {
		return zero, errors.New("index out of bounds")
	}
	if !fa.inUse[index] {
		return zero, errors.New("no item at index")
	}
	return fa.data[index], nil
}

// Remove deletes the item at index (O(1))
func (fa *FixedArray[T]) Remove(index byte) error {
	fa.lock.Lock()
	defer fa.lock.Unlock()

	if index < 0 || index >= fa.capacity {
		return errors.New("index out of bounds")
	}
	if !fa.inUse[index] {
		return errors.New("index not in use")
	}

	var zero T
	fa.data[index] = zero
	fa.inUse[index] = false
	fa.freeList = append(fa.freeList, index)
	return nil
}

func (fa *FixedArray[T]) Replace(index byte, newValue T) error {
	fa.lock.Lock()
	defer fa.lock.Unlock()

	if index < 0 || index >= fa.capacity {
		return errors.New("index out of bounds")
	}
	if !fa.inUse[index] {
		return errors.New("index not in use")
	}
	fa.data[index] = newValue
	return nil
}
