package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/miekg/dns"
)

type App struct {
	DNSRecord    string `envconfig:"DNS_RECORD" required:"true"`          // must have trailing dot
	DNSServer    string `envconfig:"DNS_SERVER" required:"true"`          // must have :53
	DNSZone      string `envconfig:"DNS_ZONE" required:"true"`            // must have trailing dot
	DNSKeyName   string `envconfig:"DNS_KEY_NAME" required:"true"`        // must have trailing dot
	DNSKeySecret string `envconfig:"DNS_KEY_Secret" required:"true"`      // no trailing dot :)
	DNSKeyAlgo   string `envconfig:"DNS_KEY_ALGO" default:"hmac-sha512."` // must have trailing dot

	Listen string `envconfig:"LISTEN" default:":8080"`
	Token  string `envconfig:"TOKEN" required:"true"`
}

func main() {
	app := &App{}
	err := envconfig.Process("", app)
	if err != nil {
		log.Fatalf("unable to parse config: %s", err)
	}

	http.HandleFunc("/update", app.handler)

	err = http.ListenAndServe(app.Listen, nil)
	if err != nil {
		log.Fatalf("unable to listen: %s", err)
	}
}

func (a *App) handler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	token := query.Get("token")
	v4 := net.ParseIP(query.Get("v4"))
	v6 := net.ParseIP(query.Get("v6"))

	if token != a.Token {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "no :(")
		return
	}
	if v4 == nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "no :(")
		return
	}

	err := a.dnsUpdate(v4, v6)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("unable to update: %s", err)
		fmt.Fprintf(w, "error")
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "updated")

}

func (a *App) dnsUpdate(v4, v6 net.IP) error {
	// dns key name as fqdn
	key := dns.Fqdn(a.DNSKeyName)

	// remove records (always includes v6)
	delRra, _ := dns.NewRR(fmt.Sprintf("%s IN A 127.0.0.1", a.DNSRecord))
	delRraaaa, _ := dns.NewRR(fmt.Sprintf("%s IN AAAA ::1", a.DNSRecord))
	delRrs := []dns.RR{delRra, delRraaaa}

	// create records
	rra := &dns.A{
		Hdr: dns.RR_Header{Name: a.DNSRecord, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
		A:   v4,
	}
	rrs := []dns.RR{rra}

	// only v6 if set
	if v6 != nil {
		rraaaa := &dns.AAAA{
			Hdr:  dns.RR_Header{Name: a.DNSRecord, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 300},
			AAAA: v6,
		}
		rrs = []dns.RR{rra, rraaaa}
	}

	// create message
	m := new(dns.Msg)
	m.SetUpdate(a.DNSZone)
	m.RemoveRRset(delRrs)
	m.Insert(rrs)
	m.SetTsig(key, a.DNSKeyAlgo, 300, time.Now().Unix())

	// client
	c := &dns.Client{
		SingleInflight: true,
		TsigSecret:     map[string]string{key: a.DNSKeySecret},
	}

	// send and handle message
	reply, _, err := c.Exchange(m, a.DNSServer)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	if reply != nil && reply.Rcode != dns.RcodeSuccess {
		return fmt.Errorf("dns update failed: %w", err)
	}

	return nil
}
