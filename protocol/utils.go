package protocol

import (
	`encoding/binary`
	`net`
)

func InetNtoA(i uint32) string {
	ip := make(net.IP, net.IPv4len)
	ip[0] = byte(i >> 24)
	ip[1] = byte(i >> 16)
	ip[2] = byte(i >> 8)
	ip[3] = byte(i)
	return ip.String()
}

func InetAtoN(ip string) uint32 {
	address := net.ParseIP(ip)
	if address == nil {
		return 0
	}

	ipv4 := address.To4()
	if ipv4 == nil {
		return 0
	}

	return binary.BigEndian.Uint32(ipv4)
}
