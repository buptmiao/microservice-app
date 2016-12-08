package util

import "net"

//GetLocalIP will return local IP address
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}
	var res string = "localhost"
	for _, address := range addrs {
		if inet, ok := address.(*net.IPNet); ok && !inet.IP.IsLoopback() {
			if inet.IP.To4() != nil {
				res = inet.IP.String()
			}
		}
	}
	return res
}
