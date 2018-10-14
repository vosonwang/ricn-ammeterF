package main

import (
	"github.com/labstack/gommon/log"
	"net"
	"strconv"
)

func HandleConn(conn net.Conn) {

	defer conn.Close()

	b := make([]byte, 512)

	readLen, err := conn.Read(b)

	if err != nil {
		log.Print("ReadError--Register", err)
		return
	}

	// 获取设备序列号
	id := b[1:readLen]

	intId, err := strconv.Atoi(string(id))

	if err != nil {
		log.Print(err)
		return
	}

	// TODO 在获取设备序列号No的时候，就应该主动通知平台有设备No上线，是否接受
	// 如果接受则开始获取数据，开始第一次询问设备，并且之后按照设置时间间隔询问设备

	// 获取设备序列号的后两位，这是约定好的通信号
	commid, err := strconv.Atoi(string(id[len(id)-2:]))

	if err != nil {
		log.Print(err)
		return
	}

	log.Printf("序列号：%d,通信ID：%d", intId, commid)

	device := new(Device)

	// 开始获取设备上的各项数据

	/*外接 CT(穿刺） 类型*/
	ct, err := GetData(conn, commid, 6008, 2, "uint32")

	if err != nil {
		log.Print("获取外接 CT(穿刺） 类型失败：", err)
		return
	}

	//log.Print(ct)

	device.CT = ct.([]uint32)[0]

	/*
		0-A相电流
		1-B相电流、
		2-C相电流、
		3-剩余电流、
		4-A相温度/温度1、
		5-B相温度/温度2、
		6-C相温度/温度3、
		7-箱体温度/温度4
	*/
	currentAndTemperature, err := GetData(conn, commid, 40004, 16, "float32")

	if err != nil {
		log.Print("获取电流、温度失败：", err)
		return
	}

	// log.Print(currentAndTemperature)  // [0 0.536708 1.3579785 32767  22.30535 21.697397 21.545958 22.361412]

	device.Ia = currentAndTemperature.([]float32)[0]
	device.Ib = currentAndTemperature.([]float32)[1]
	device.Ic = currentAndTemperature.([]float32)[2]
	device.IR = currentAndTemperature.([]float32)[3]
	device.T1 = currentAndTemperature.([]float32)[4]
	device.T2 = currentAndTemperature.([]float32)[5]
	device.T3 = currentAndTemperature.([]float32)[6]
	device.T4 = currentAndTemperature.([]float32)[7]

	/*
		0-A相电压
		1-B相电压
		2-C相电压
		4-总功率因数
		5-三相有功功率
	*/

	voltageAndPower, err := GetData(conn, commid, 40020, 6, "uint16")

	if err != nil {
		log.Print("获取电压、功率失败：", err)
		return
	}
	// log.Print(voltageAndPower) //  [23494 23517 4833 5003 9834 5]

	device.Ua = float32(voltageAndPower.([]uint16)[0]) * 0.01
	device.Ub = float32(voltageAndPower.([]uint16)[1]) * 0.01
	device.Uc = float32(voltageAndPower.([]uint16)[2]) * 0.01
	device.Pf = float32(voltageAndPower.([]uint16)[4]) * 0.0001
	device.Ptot = float32(voltageAndPower.([]uint16)[5]) * 0.01

	/*
		0-DO1
		1-DO2
	*/
	dos, err := GetData(conn, commid, 40039, 2, "uint16")

	if err != nil {
		log.Print("获取DO1、DO2失败：", err)
		return
	}

	// log.Print(dos) // [0 0]

	device.DO1 = dos.([]uint16)[0]
	device.DO2 = dos.([]uint16)[1]

	/*
		0-A 相电流限值（定值 越限 1）
		1-B 相电流限值（定值 越限 2）
		2-C 相电流限值（定值 越限 3）
		8-DO1 控制寄存器 （定值越限 9）
		9-DO2控制寄存器
	*/

	ISetting, err := GetData(conn, commid, 40042, 20, "float32")

	if err != nil {
		log.Print("获取电流限值失败：", err)
		return
	}

	//log.Print(ISetting)

	device.IaSetting = ISetting.([]float32)[0]
	device.IbSetting = ISetting.([]float32)[1]
	device.IcSetting = ISetting.([]float32)[2]
	device.DO1Setting = ISetting.([]float32)[8]
	device.DO2Setting = ISetting.([]float32)[9]

	/*校时*/
	//if err := Timing(conn, commid); err != nil {
	//	log.Print("校时失败", err)
	//	return
	//}

	/*获取装置时间*/
	device.DeviceTime, err = GetDeviceTime(conn, commid)

	if err != nil {
		log.Print("获取时间失败：", err)
		return
	}

	/*
		烟雾报警-42010的bit0
		柜门状态-42010的bit1
	*/
	DIDOStatus, err := GetData(conn, commid, 42010, 1, "uint16")

	if err != nil {
		log.Print("获取正向有功电能失败：", err)
		return
	}
	//log.Print(DIDOStatus)

	device.DI1 = DIDOStatus.([]uint16)[0] >> 0 & 1 // 取42010的bit0
	device.DI1 = DIDOStatus.([]uint16)[0] >> 1 & 1 // 取42010的bit1

	/*
		缺相	42011的bit11
		过压	42011的bit8
		欠压	42011的bit9
	*/
	totalAlarmStatus, err := GetData(conn, commid, 42011, 1, "uint16")

	if err != nil {
		log.Print("获取正向有功电能失败：", err)
		return
	}

	// log.Print(totalAlarmStatus)

	device.PhaseDeficiencyProtection = totalAlarmStatus.([]uint16)[0] >> 11 & 1
	device.OverVoltageProtection = totalAlarmStatus.([]uint16)[0] >> 8 & 1
	device.UnderVoltageProtection = totalAlarmStatus.([]uint16)[0] >> 9 & 1

	/*正向有功电能*/

	ImpKwh, err := GetData(conn, commid, 42116, 2, "uint32")

	if err != nil {
		log.Print("获取正向有功电能失败：", err)
		return
	}

	device.ImpKwh = float32(ImpKwh.([]uint32)[0]) * 0.01

	/*
		0-报警声音
		1-故障声音
		2-告警复归方式
	*/
	sounds, err := GetData(conn, commid, 50000, 3, "uint16")

	if err != nil {
		log.Print("获取报警声音、故障声音、告警复归方式失败：", err)
		return
	}

	//log.Print(sounds)

	device.AlarmSound = sounds.([]uint16)[0]
	device.FaultSound = sounds.([]uint16)[1]
	device.Reset = sounds.([]uint16)[2]

	/*
		0-IR1 报警设定值
		5-TC1 报警设定值
		10-TC2 报警设定值
		15-TC3 报警设定值
		20-TC4 报警设定值
	*/
	alarmSetting, err := GetData(conn, commid, 50021, 21, "uint16")

	if err != nil {
		log.Print("获取报警设定值失败：", err)
		return
	}

	// log.Print("报警设定值",alarmSetting)

	device.IRSetting = alarmSetting.([]uint16)[0]
	device.T1Setting = alarmSetting.([]uint16)[5]
	device.T2Setting = alarmSetting.([]uint16)[10]
	device.T3Setting = alarmSetting.([]uint16)[15]
	device.T4Setting = alarmSetting.([]uint16)[20]

	gprs, err := GetData(conn, commid, 140, 2, "uint16")

	if err != nil {
		log.Print("获取gprs失败：", err)
		return
	}

	// log.Print("gprs: ", gprs)

	device.GPRSRSSI = gprs.([]uint16)[0]
	device.GPRSOperator = gprs.([]uint16)[1]

	/*
		昨日电量 示例： 3.43
		0-EnergyYesterday
	*/

	// 烟雾报警 正常
	// 柜门状态 正常
	// 空开状态 正常
	// 报警器  正常
	// 缺相 正常
	// 欠压 正常
	// 过压 正常

	log.Printf("A相电流：%vA ; B相电流 ：%vA ;C相电流 : %vA;剩余电流：%vmA;A相温度：%v°C;B相温度：%v°C;C相温度：%v°C;箱体温度：%v°C;A相电压：%vV;B相电压：%vV;C相电压：%vV;总功率因数：%v;三相有功功率：%vkW;空开状态：%v;报警器：%v;A相电流限值：%vA;B相电流限值：%vA;C相电流限值：%vA;DO1控制寄存器：%v;DO2控制寄存器：%v;装置时间：%v;正向有功电能总和：%vkWh;烟雾报警：%v;柜门状态：%v;穿刺：%v;报警声音：%v;故障声音：%v;TC1报警设定值：%v℃;TC2报警设定值：%v℃;TC3报警设定值：%v℃;TC4报警设定值：%v℃;IR报警设定值：%vmA;缺相：%v;过压：%v;欠压：%v;GPRS信号强度：%v;GPRS运营商：%v;告警复归方式：%v", device.Ia, device.Ib, device.Ic, device.IR, device.T1, device.T2, device.T3, device.T4, device.Ua, device.Ub, device.Uc, device.Pf, device.Ptot, device.DO1, device.DO2, device.IaSetting, device.IbSetting, device.IcSetting, device.DO1Setting, device.DO2Setting, device.DeviceTime, device.ImpKwh, device.DI1, device.DI2, device.CT, device.AlarmSound, device.FaultSound, device.T1Setting, device.T2Setting, device.T3Setting, device.T4Setting, device.IRSetting, device.PhaseDeficiencyProtection, device.OverVoltageProtection, device.UnderVoltageProtection, device.GPRSRSSI, device.GPRSOperator, device.Reset)

}
