package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
)

const UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36 Edg/137.0.0.0"

type Address struct {
	Ip string `json:"ip"`
}

func requestIp(is_v6 bool) (string, error) {
	var url string

	if is_v6 {
		url = "https://6.ipw.cn"
	} else {
		url = "https://4.ipw.cn"
	}
	fmt.Println("Requesting", url)

	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", UA)
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Request", url, "error:", err)
	} else {
		defer res.Body.Close()
		if res.StatusCode == 200 {
			resBody, err := io.ReadAll(res.Body)
			if err == nil {
				return string(resBody), nil
			}
		}
	}

	err = nil
	if is_v6 {
		url = "https://ipv6.jsonip.com"
	} else {
		url = "https://ipv4.jsonip.com"
	}
	fmt.Println("Requesting", url)

	req, _ = http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", UA)
	res, err = client.Do(req)
	if err != nil {
		return "", err
	}
	if res.StatusCode != 200 {
		return "", fmt.Errorf("%s statusCode: %d", url, res.StatusCode)
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var jo Address
	if err := json.Unmarshal(resBody, &jo); err != nil {
		return "", err
	}

	return jo.Ip, nil
}

func main() {
	var region string
	var ak string
	var sk string
	var dn string
	var rr string
	var t string
	var v string
	var show_myip bool

	flag.StringVar(&region, "region", "cn-hangzhou", "Region")
	flag.StringVar(&ak, "ak", os.Getenv("AK"), "AK")
	flag.StringVar(&sk, "sk", os.Getenv("SK"), "SK")
	flag.StringVar(&dn, "dn", "umutech.com", "DomainName")
	flag.StringVar(&rr, "rr", "umu618", "RR")
	flag.StringVar(&t, "t", "A", "Type")
	flag.StringVar(&v, "v", "", "Value")
	flag.BoolVar(&show_myip, "ip", false, "Show my IP")

	flag.Parse()

	if t == "" {
		fmt.Println("Error: no Type!")
		return
	}
	fmt.Printf("Type: %s\n", t)

	is_v6 := false
	switch t {
	case "A":
		break
	case "AAAA":
		is_v6 = true
		break
	default:
		fmt.Println("Error: bad Type %s", t)
		return
	}

	if show_myip {
		ip, err := requestIp(is_v6)
		if err == nil {
			fmt.Println(ip)
		} else {
			fmt.Println("Error: making http request: %s", err)
		}
		return
	}

	if ak == "" {
		fmt.Println("Error: no AK!")
		return
	}
	// fmt.Printf("AK: %s\n", ak)
	if sk == "" {
		fmt.Println("Error: no SK!")
		return
	}
	if dn == "" {
		fmt.Println("Error: no DomainName!")
		return
	}
	fmt.Printf("DomainName: %s\n", dn)
	if rr == "" {
		fmt.Println("Error: no RR!")
		return
	}
	fmt.Printf("RR: %s\n", rr)

	if v == "" {
		ip, err := requestIp(is_v6)
		if err != nil {
			fmt.Println("Error: making http request: %s", err)
			return
		}
		v = ip
	}
	fmt.Printf("Value: %s\n", v)

	client, err := alidns.NewClientWithAccessKey(region, ak, sk)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return
	}

	desc := alidns.CreateDescribeDomainRecordsRequest()
	desc.DomainName = dn
	desc.SearchMode = "EXACT"
	desc.KeyWord = rr
	existed, err := client.DescribeDomainRecords(desc)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return
	}

	if existed.TotalCount == 0 {
		request := alidns.CreateAddDomainRecordRequest()
		request.Scheme = "https"
		request.DomainName = dn
		request.RR = rr
		request.Type = t
		request.Value = v
		response, err := client.AddDomainRecord(request)
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			return
		}
		fmt.Printf("OK: %s\n", response)
	} else {
		fmt.Printf("TotalCount: %#v\n", existed.TotalCount)
		var rid string = ""
		for _, r := range existed.DomainRecords.Record {
			if r.RR == rr {
				if r.Type == t && r.Value == v {
					fmt.Println("No change!")
					return
				}
				rid = r.RecordId
				break
			}
		}
		if len(rid) == 0 {
			fmt.Printf("Error: %s not found!\n", rr)
			return
		}

		request := alidns.CreateUpdateDomainRecordRequest()
		request.Scheme = "https"
		request.RecordId = rid
		request.RR = rr
		request.Type = t
		request.Value = v
		response, err := client.UpdateDomainRecord(request)
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			return
		}
		fmt.Printf("OK: %s\n", response)
	}
}
