package webrtc

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"

	"math"
	"math/big"
)

const sliceSize int = 16 * 1024

///把二进制消息分片和汇总
type MessageSlice struct {
	sliceBuffer map[int][]byte

	sliceBufferId int
}

func (this *MessageSlice) slice(message []byte) map[int][]byte {
	total := len(message)
	remainder := total % sliceSize
	sliceCount := total / sliceSize
	if remainder > 0 {
		sliceCount++
	}
	randomNum, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt))
	slices := make(map[int][]byte)
	for i := 0; i < sliceCount; i++ {
		buffer := bytes.NewBuffer([]byte{})
		binary.Write(buffer, binary.BigEndian, int(randomNum.Uint64()))
		binary.Write(buffer, binary.BigEndian, sliceCount)
		binary.Write(buffer, binary.BigEndian, i)
		start := i * sliceSize
		end := (i + 1) * sliceSize
		if end < total {
			buffer.Write(message[start:end])
		} else {
			buffer.Write(message[start:])
		}
		slices[i] = buffer.Bytes()
	}

	return slices
}

func (this *MessageSlice) merge(data []byte) []byte {
	buffer := bytes.NewBuffer(data[0:4])
	var id int
	binary.Read(buffer, binary.BigEndian, &id)
	if id != this.sliceBufferId {
		this.sliceBufferId = id
		this.sliceBuffer = make(map[int][]byte)
	}
	buffer = bytes.NewBuffer(data[4:8])
	var sliceCount int
	binary.Read(buffer, binary.BigEndian, &sliceCount)
	buffer = bytes.NewBuffer(data[8:12])
	var i int
	binary.Read(buffer, binary.BigEndian, &i)
	if sliceCount == 1 {
		this.sliceBufferId = 0
		this.sliceBuffer = make(map[int][]byte)
		return data[12:]
	} else {
		sliceData, ok := this.sliceBuffer[i]
		if !ok {
			this.sliceBuffer[i] = data
		}

		sliceBufferSize := len(this.sliceBuffer)
		if sliceBufferSize == sliceCount {
			buffer = bytes.NewBuffer([]byte{})
			for j := 0; j < sliceBufferSize; j++ {
				sliceData, ok = this.sliceBuffer[j]
				if ok {
					buffer.Write(sliceData[12:])
				}
			}

			this.sliceBufferId = 0
			this.sliceBuffer = make(map[int][]byte)

			return buffer.Bytes()
		}
	}
	return nil
}
