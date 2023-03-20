# webhook-rfc2136

webhook-rfc2136 is a very small glue between http webhooks and rfc2136 dns updates. It's meant to allow a Fritzbox to update self hosted bind DNS Servers.

## bind configuration

The secret key can be gerneated with `rndc-confgen -A hmac-sha512 -k home-key`.

```
# in the top level config
key "dyn-key" {
  algorithm hmac-sha512;
  secret "<dyn-key-secret>";
};

# within a zone section
zone "example.org" in {
  type master;
  file "example.org.zone";

  update-policy {
    grant dyn-key name dyn.example.org. A AAAA;
  };
};
```

## docker container

Note: all the trailing dots are required.

```
docker run -it \
  -e DNS_SERVER=dns.example.org:53 \
  -e DNS_KEY_NAME=dyn-key. \
  -e DNS_KEY_SECRET="<dyn-key-secret>" \
  -e TOKEN="<webhook-secret>" \
  -e DNS_RECORD=dyn.example.org. \
  foosinn/webhook-rfc2136
```

## fritzbox

* Internet -> Freigaben
* Reiter: DynDNS
* [x] DynDNS benutzen

Werte:

* DynDNS-Anbieter: Benutzerdefiniert
* Update-URL: https://rfc2136.example.org/update?token=<username>&v4=<ipaddr>&v6=<ip6addr>
* Domainname: does-not-matter
* Benutzername: <webhook-secret>
* Kennwort: does-not-matter
