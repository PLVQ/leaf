package network

import (
	"encoding/binary"
	"errors"
	"io"
	"math"

)

type MsgParser struct {
	lenMsgLen    int // 消息长度数据所占字节长度
	minMsgLen    uint32
	maxMsgLen    uint32
	littleEndian bool
}

func NewMsgParser() *MsgParser {
	msgParser := new(MsgParser)
	msgParser.lenMsgLen = 2
	msgParser.minMsgLen = 1
	msgParser.maxMsgLen = 4096
	msgParser.littleEndian = false

	return msgParser
}

func (msg *MsgParser) SetMsgLen(lenMsgLen int, minMsgLen uint32, maxMsgLen uint32) {
	if lenMsgLen == 1 || lenMsgLen == 2 || lenMsgLen == 4 {
		msg.lenMsgLen = lenMsgLen
	}

	if minMsgLen != 0 {
		msg.minMsgLen = minMsgLen
	}

	if maxMsgLen != 0 {
		msg.maxMsgLen = maxMsgLen
	}

	var max uint32
	switch msg.lenMsgLen {
	case 1:
		max = math.MaxUint8
	case 2:
		max = math.MaxUint16
	case 4:
		max = math.MaxUint32
	}

	if msg.minMsgLen > max {
		msg.minMsgLen = max
	}

	if msg.maxMsgLen > max {
		msg.maxMsgLen = max
	}
}

func (msg *MsgParser) SetByteOrder(littleEndian bool) {
	msg.littleEndian = littleEndian
}

func (msg *MsgParser) Read(conn *TCPConn, args ...[]byte) ([]byte, error) {

	var b [4]byte

	// 消息长度
	bufMsgLen := b[:msg.lenMsgLen]

	if _, err := io.ReadFull(conn, bufMsgLen); err != nil {
		return nil, err
	}

	var msgLen uint32
	switch msg.lenMsgLen {
	case 1:
		msgLen = uint32(bufMsgLen[0])
	case 2:
		if msg.littleEndian {
			msgLen = uint32(binary.LittleEndian.Uint16(bufMsgLen))
		} else {
			msgLen = uint32(binary.BigEndian.Uint16(bufMsgLen))
		}
	case 4:
		if msg.littleEndian {
			msgLen = binary.LittleEndian.Uint32(bufMsgLen)
		} else {
			msgLen = binary.BigEndian.Uint32(bufMsgLen)
		}
	}

	if msgLen > msg.maxMsgLen {
		return nil, errors.New("Message too long")
	} else if msgLen < msg.minMsgLen {
		return nil, errors.New("message too short")
	}

	msgData := make([]byte, msgLen)
	if _, err := io.ReadFull(conn, msgData); err != nil {
		return nil, err
	}

	return msgData, nil
}

func (msg *MsgParser) Write(conn *TCPConn, args ...[]byte) error {
	var msgLen uint32
	for i := 0; i < len(args); i++ {
		msgLen += uint32(len(args[i]))
	}

	if msgLen > msg.maxMsgLen {
		return errors.New("Message too long")
	} else if msgLen < msg.minMsgLen {
		return errors.New("Message too short")
	}

	mssage := make([]byte, uint32(msg.lenMsgLen)+msgLen)

	switch msg.lenMsgLen {
	case 1:
		mssage[0] = byte(msgLen)
	case 2:
		if msg.littleEndian {
			binary.LittleEndian.PutUint16(mssage, uint16(msgLen))
		} else {
			binary.BigEndian.PutUint16(mssage, uint16(msgLen))
		}
	case 4:
		if msg.littleEndian {
			binary.LittleEndian.PutUint32(mssage, uint32(msgLen))
		} else {
			binary.BigEndian.PutUint32(mssage, uint32(msgLen))
		}
	}

	l := msg.lenMsgLen
	for i := 0; i < len(args); i++ {
		copy(mssage[l:], args[i])
		l += len(args[i])
	}

	conn.Write(mssage)

	return nil
}
