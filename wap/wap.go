package wap

import (
	"errors"

	"github.com/psanford/gsm/mms"
)

var invalidPacket = errors.New("invalid push notification wap packet")

func UnmarshalPushNotification(packet []byte) (*mms.Message, error) {
	if len(packet) < 6 {
		return nil, invalidPacket
	}
	wapPush := packet[1]

	if wapPush != 0x06 {
		return nil, invalidPacket
	}

	offset := packet[2]
	offset += 3

	if int(offset) >= len(packet) {
		return nil, invalidPacket
	}

	packet = packet[offset:]

	return mms.Unmarshal(packet)
}
