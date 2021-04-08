package main

import (
	"flag"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

type IPing struct {
	Host   string
	Ip     string
	IpType int
	Args   []string
	Ttl    string
}

type IPingTask struct {
	OsType string
	Count  string
	IPings []IPing
}

func newTask() *IPingTask {
	//get os and set default value
	return &IPingTask{OsType: getOsType(), Count: "4"}
}

func (i *IPingTask) addTask(ip string) {
	var iping IPing
	//select ip or host, then set ipv4 or ipv6
	if ipaddr := net.ParseIP(ip); ipaddr != nil {
		iping.Ip = ip
		iping.IpType = 6
		if ipaddr.To4() != nil {
			iping.IpType = 4
		}
	} else {
		iping.Host = ip
	}

	i.IPings = append(i.IPings, iping)
}

func (i *IPingTask) setCount(c string) {
	//default value is 4, set in init
	if c != "" {
		i.Count = c
	}
}

func (i *IPingTask) print() {
	for _, v := range i.IPings {
		if v.Ttl != "" {
			fmt.Println(v.Host, v.Ip, v.Ttl)
		}
	}
}

func (i *IPing) setArgs(o, c string) {
	var args []string
	if i.Host != "" {
		args = append(args, i.Host)
	} else {
		args = append(args, i.Ip)
		if i.IpType == 4 {
			args = append(args, "-4")
		} else {
			args = append(args, "-6")
		}
	}

	if o == "linux" {
		args = append(args, "-c", c)
	} else {
		args = append(args, "-n", c)
	}
	i.Args = args
}

func (i *IPing) pingTest() {
	out, _ := exec.Command("ping", i.Args...).Output()

	if strings.Contains(string(out), "TTL") {
		re := regexp.MustCompile(`TTL=(.*)`)
		i.Ttl = fmt.Sprintf("%q", re.FindAll(out, 1))
	} else {
		re := regexp.MustCompile(`ttl=(.*)`)
		i.Ttl = fmt.Sprintf("%q", re.FindAll(out, 1))
	}
	//if strings.Contains(string(out), "TTL") || strings.Contains(string(out), "ttl") {
	//	i.Ttl = "123"
	//}
}

func getOsType() string {
	return runtime.GOOS
}

func incIp(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func foreachIp(netCIDR string) (ips []string) {
	ip, ipNet, err := net.ParseCIDR(netCIDR)
	if err != nil {
		fmt.Println("invalid CIDR")
	}

	for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); incIp(ip) {
		ips = append(ips, ip.String())
	}
	return
}

var (
	ips []string
	wg  sync.WaitGroup
)

func main() {
	t := flag.String("t", "", "host or ip addr, i.e.: \n-t 127.0.0.1 or -t www.baidu.com")
	i := flag.String("i", "", "ip addr of range, i.e.: \n-i 127.0.0.1/24")
	c := flag.String("c", "4", "request arp packet count, i.e.: \n-c 4")
	flag.Parse()

	if *t == "" && *i == "" {
		flag.Usage()
		return
	}

	result := make(chan IPing)
	task := newTask()
	task.setCount(*c)

	if *i != "" {
		ips = foreachIp(*i)
	} else {
		ips = append(ips, *t)
	}

	for _, v := range ips {
		task.addTask(v)
	}

	for _, v := range task.IPings {
		wg.Add(1)
		go func(i IPing) {
			defer wg.Done()
			i.setArgs(task.OsType, task.Count)
			i.pingTest()
			if i.Ttl != "" {
				result <- i
			}
		}(v)
	}
	go func() {
		for v := range result {
			fmt.Println(v.Host, v.Ip, v.Ttl)
		}
	}()
	wg.Wait()
	time.Sleep(2)
}
