package bigqueue

import (
	"errors"
)

const (
	cInt64Size = 8
)

var (
	// ErrEmptyQueue is returned when peek/dequeue is performed on an empty queue
	ErrEmptyQueue = errors.New("queue is empty")
)

// Peek returns the head of the queue
func (bq *BigQueue) Peek() ([]byte, error) {
	return bq.getQueueHead(false)
}

// Dequeue removes an element from the queue
func (bq *BigQueue) Dequeue() ([]byte, error) {
	return bq.getQueueHead(true)
}

// getQueueHead gets the head of the queue and deletes the head if dequeue is true
func (bq *BigQueue) getQueueHead(dequeue bool) ([]byte, error) {
	if bq.IsEmpty() {
		return nil, ErrEmptyQueue
	}

	// read index
	aid, offset := bq.index.getHead()

	// read length
	var length int
	aid, offset, length = bq.readLength(aid, offset)

	// read message
	aid, offset, message, err := bq.readBytes(aid, offset, length)
	if err != nil {
		return nil, err
	}

	// update head
	if dequeue {
		bq.index.putHead(aid, offset)
	}

	return message, nil
}

// readLength reads length of the message
func (bq *BigQueue) readLength(aid, offset int) (int, int, int) {
	// check if length is present in same arena, if not get next arena.
	// If length is stored in next arena, get next aid with 0 offset value
	if offset+cInt64Size > bq.arenaSize {
		aid, offset = aid+1, 0
	}

	length := int(bq.arenaList[aid].ReadUint64(offset))
	offset += cInt64Size

	return aid, offset, length
}

// readBytes reads length bytes from arena aid starting at offset, if length
// is bigger than arena size, it calls readBytesFromMultipleArenas
func (bq *BigQueue) readBytes(aid, offset, length int) (int, int, []byte, error) {
	byteSlice := make([]byte, length)

	// check if length can be read from same arena
	if offset+length <= bq.arenaSize {
		if _, err := bq.arenaList[aid].Read(byteSlice, offset); err != nil {
			return 0, 0, nil, err
		}

		offset += length
	} else {
		var err error
		aid, offset, err = bq.readBytesFromMultipleArenas(aid, offset, byteSlice)
		if err != nil {
			return 0, 0, nil, err
		}
	}

	return aid, offset, byteSlice, nil
}

// readBytesFromMultipleArenas is called when length to be read is greater than arena size
func (bq *BigQueue) readBytesFromMultipleArenas(aid, offset int, byteSlice []byte) (
	int, int, error) {

	counter := 0
	for {
		bytesRead, err := bq.arenaList[aid].Read(byteSlice[counter:], offset)
		if err != nil {
			return 0, 0, err
		}
		counter += bytesRead

		if counter < len(byteSlice) {
			aid, offset = aid+1, 0
		} else {
			offset = bytesRead
			break
		}
	}

	return aid, offset, nil
}
