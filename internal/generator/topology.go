package generator

type Topology struct {
	Frontdoor string
	Services  []string
}

func BuildTopology(prefix string, n int) Topology {
	services := make([]string, 0, n)
	services = append(services, "api-gateway")
	for i := 1; i < n; i++ {
		services = append(services, prefix+itoa(i))
	}
	return Topology{
		Frontdoor: services[0],
		Services:  services,
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + (v % 10))
		v /= 10
	}
	return string(buf[i:])
}
