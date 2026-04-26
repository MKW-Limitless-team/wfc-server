package common

import "encoding/binary"

// References:
// https://wiibrew.org/wiki/Mii_Data
// https://github.com/kiwi515/ogws/tree/master/src/RVLFaceLib

type Mii [0x4C]byte

type RawMii struct {
	Data [0x4C]byte
}

func RawMiiFromBytes(data []byte) RawMii {
	var miiData [0x4C]byte
	copy(miiData[:], data[:0x4C])
	return RawMii{Data: miiData}
}

func (data Mii) RFLCalculateCRC() uint16 {
	crc := uint16(0)

	for _, val := range data {
		for j := 0; j < 8; j++ {
			if crc&0x8000 != 0 {
				crc <<= 1
				crc ^= 0x1021
			} else {
				crc <<= 1
			}

			if val&0x80 != 0 {
				crc ^= 0x1
			}

			val <<= 1
		}
	}

	return crc
}

func (data RawMii) CalculateMiiCRC() uint16 {
	crc := uint16(0)

	for _, val := range data.Data {
		for j := 0; j < 8; j++ {
			if crc&0x8000 != 0 {
				crc <<= 1
				crc ^= 0x1021
			} else {
				crc <<= 1
			}

			if val&0x80 != 0 {
				crc ^= 0x1
			}

			val <<= 1
		}
	}

	return crc
}

var officialMiiList = []uint64{
	0x80000000ECFF82D2,
	0x80000001ECFF82D2,
	0x80000002ECFF82D2,
	0x80000003ECFF82D2,
	0x80000004ECFF82D2,
	0x80000005ECFF82D2,
}

func RFLSearchOfficialData(id uint64) (bool, int) {
	for i, v := range officialMiiList {
		if v == id {
			return true, i
		}
	}

	return false, -1
}

func SearchOfficialMiiData(id uint64) (bool, int) {
	return RFLSearchOfficialData(id)
}

// ClearMiiInfo clears Mii fields that should not be exposed publicly.
func (mii RawMii) ClearMiiInfo() RawMii {
	if found, _ := SearchOfficialMiiData(binary.BigEndian.Uint64(mii.Data[0x18:0x20])); found {
		return mii
	}

	binary.BigEndian.PutUint32(mii.Data[0x18:0x1C], 0x80000000)
	binary.BigEndian.PutUint32(mii.Data[0x1C:0x20], 0)

	hitNullTerminator := false
	for i := 0; i < 20; i += 2 {
		if hitNullTerminator {
			mii.Data[0x2+i] = 0
			mii.Data[0x2+i+1] = 0
		} else if mii.Data[0x2+i] == 0 && mii.Data[0x2+i+1] == 0 {
			hitNullTerminator = true
		}
	}

	for i := 0; i < 20; i++ {
		mii.Data[0x36+i] = 0
	}

	mii.Data[0] &= ^byte(0x3F)
	mii.Data[1] &= ^byte(0xE0)

	mii.Data[0x4A] = 0
	mii.Data[0x4B] = 0
	crc := mii.CalculateMiiCRC()
	mii.Data[0x4A] = byte(crc >> 8)
	mii.Data[0x4B] = byte(crc & 0xFF)

	return mii
}
