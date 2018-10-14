package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/labstack/gommon/log"
	"net"
	"ricnsmart.com/ammeterF/mbserver"
	"time"
)

// 写寄存器、发送下行报文
func Send(conn net.Conn, Commid int, start uint16, num uint) (err error) {

	frame := &mbserver.RTUFrame{
		Address:  uint8(Commid),
		Function: uint8(0x03),
		Data:     []byte{},
	}

	mbserver.SetDataWithRegisterAndNumber(frame, start, uint16(num))

	/* 设置发送超时 */
	conn.SetWriteDeadline(time.Now().Add(time.Second * 30))

	// log.Printf("发送的报文：%v", frame.Bytes())

	/* 发送数据 */
	_, err = conn.Write(frame.Bytes())

	if err != nil {
		log.Print("SendError ", err)
		return
	}

	return nil

}

// 写寄存器，发送下行报文
func Write(conn net.Conn, Commid int, start, num uint16, values []int) (err error) {

	var buf bytes.Buffer

	for _, value := range values {
		binary.Write(&buf, binary.BigEndian, float32(value))
	}

	frame := &mbserver.RTUFrame{
		Address:  uint8(Commid),
		Function: uint8(0x10),
		Data:     []byte{},
	}

	mbserver.SetDataWithRegisterAndNumberAndBytes(frame, start, num, buf.Bytes())

	/* 设置发送超时 */
	conn.SetWriteDeadline(time.Now().Add(time.Second * 30))

	// log.Printf("发送的报文：%v", frame.Bytes())

	/* 发送数据 */
	_, err = conn.Write(frame.Bytes())

	if err != nil {
		log.Print("SendError ", err)
		return
	}

	return nil

}

// 读取上行报文
func Read(conn net.Conn) (b []byte, err error) {
	b = make([]byte, 512)

	readLen, err := conn.Read(b)

	if err != nil {
		log.Print(err)
		return
	}

	b = b[:readLen]

	// log.Printf("接收到的报文：%v", b)

	return
}

// 验证读寄存器返回的报文
func ValidateRead(Commid int, recvs []byte, regnum uint) (rtuFrame *mbserver.RTUFrame, err error) {
	frame, e0 := mbserver.NewRTUFrame(recvs)
	if e0 != nil {
		err = errors.New("读寄存器 crc校验错误")
		return
	}

	/* 校验通信ID */
	if uint8(Commid) != frame.Address {
		err = errors.New("读寄存器 通信ID 错误")
		return
	}

	//log.Print(frame.GetFunction() )

	/* 校验功能码 */
	if frame.GetFunction() != 0x03 {
		err = errors.New("读寄存器 功能码 错误")
		return
	}

	/* 校验回复帧长度 */
	if len(frame.GetData()) != (int)(regnum*2+1) {
		err = errors.New("读寄存器 帧长度 错误")
		return
	}

	/* 校验字节数 */
	if frame.GetData()[0] != (byte)(regnum*2) {
		err = errors.New("读寄存器 字节数 错误")
		return
	}

	rtuFrame = frame

	return
}

// 验证写寄存器返回的报文
func ValidateWrite(Commid int, recvs []byte, start uint16, regnum uint) (err error) {
	frame, e0 := mbserver.NewRTUFrame(recvs)
	if e0 != nil {
		err = errors.New("写寄存器 crc校验 error!")
		return
	}

	/* 校验通信ID */
	if uint8(Commid) != frame.Address {
		err = errors.New("写寄存器 验通信ID 错误")
		return
	}

	//log.Print(frame.GetFunction() )

	/* 校验功能码 */
	if frame.GetFunction() != 0x10 {
		err = errors.New("写寄存器 功能码 错误")
		return
	}

	data := frame.GetData()

	/* 校验回复帧长度 */
	if len(data) != 4 {
		err = errors.New("写寄存器 帧长度 错误")
		return
	}

	// 校验寄存器地址
	if binary.BigEndian.Uint16(data[0:2]) != start {
		err = errors.New("写寄存器 寄存器地址 错误")
		return
	}
	// 校验寄存器数量
	if binary.BigEndian.Uint16(data[2:4]) != uint16(regnum) {
		err = errors.New("写寄存器 寄存器数量 错误")
		return
	}

	return
}

// 读寄存器后从上行的报文帧中解析数据
func Parse(frame *mbserver.RTUFrame, num uint, dataType string) (interface{}, error) {
	/* 解析数据,  */
	data := frame.GetData()[1:]
	switch dataType {
	// uint16 int16
	case "uint16":
		return mbserver.DecodeUint16s(&data, num)
		// int32
	case "uint32":
		return mbserver.DecodeUint32s(&data, num)
		// float
	case "float32":
		return mbserver.DecodeFloat32s(&data, num)
	default:
		return nil, errors.New("类型错误")
	}
}

func GetData(conn net.Conn, commid int, start uint16, num uint, dataType string) (data interface{}, err error) {

	b := make([]byte, 512)

	if err = Send(conn, commid, start, num); err != nil {
		return
	}

	if b, err = Read(conn); err != nil {
		return
	}

	frame, err := ValidateRead(commid, b, num)

	if err != nil {
		return
	}

	switch dataType {
	case "float32", "uint32":
		data, err = Parse(frame, num/2, dataType)
	case "uint16":
		data, err = Parse(frame, num, dataType)
	}

	if err != nil {
		return
	}

	return
}
