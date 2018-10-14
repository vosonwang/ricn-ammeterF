package main

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

type Device struct {
	Ia                        float32
	Ib                        float32
	Ic                        float32
	IR                        float32
	T1                        float32
	T2                        float32
	T3                        float32
	T4                        float32
	Ua                        float32
	Ub                        float32
	Uc                        float32
	Pf                        float32
	Ptot                      float32
	DO1                       uint16
	DO2                       uint16
	IaSetting                 float32
	IbSetting                 float32
	IcSetting                 float32
	DO1Setting                float32
	DO2Setting                float32
	DeviceTime                string
	ImpKwh                    float32
	DI1                       uint16
	DI2                       uint16
	CT                        uint32
	AlarmSound                uint16
	FaultSound                uint16
	T1Setting                 uint16
	T2Setting                 uint16
	T3Setting                 uint16
	T4Setting                 uint16
	IRSetting                 uint16
	PhaseDeficiencyProtection uint16
	OverVoltageProtection     uint16
	UnderVoltageProtection    uint16
	GPRSRSSI                  uint16
	GPRSOperator              uint16
	Reset                     uint16
	EnergyYesterday           uint32
}

// 获取设备当前时间
func GetDeviceTime(conn net.Conn, commid int) (deviceTime string, err error) {

	b := make([]byte, 512)

	/*
		装置时间（年）
		装置时间（月）
		装置时间（日）
		装置时间（时）
		装置时间（分）
		装置时间（秒）
	*/

	if err = Send(conn, commid, 40062, 12); err != nil {
		return
	}

	if b, err = Read(conn); err != nil {
		return
	}

	frame, err := ValidateRead(commid, b, 12)

	if err != nil {
		return
	}

	Nos, err := Parse(frame, 6, "float32")

	if err != nil {
		return
	}

	// log.Print(Nos) // [18 9 9 23 45 55]

	deviceTime = fmt.Sprintf("%v%v-%v-%v %v:%v:%v", time.Now().Format("2006")[0:2], Nos.([]float32)[0], Nos.([]float32)[1], Nos.([]float32)[2], Nos.([]float32)[3], Nos.([]float32)[4], Nos.([]float32)[5])

	return
}

// 设备校时
func Timing(conn net.Conn, commid int) (err error) {

	now := time.Now()

	year, _ := strconv.Atoi(now.Format("06"))

	month, _ := strconv.Atoi(now.Format("01"))

	b := make([]byte, 512)

	if err = Write(conn, commid, 40062, 12, []int{year, month, now.Day(), now.Hour(), now.Minute(), now.Second()}); err != nil {
		return
	}

	if b, err = Read(conn); err != nil {
		return
	}

	if err = ValidateWrite(commid, b, 40062, 12); err != nil {
		return
	}

	return
}
