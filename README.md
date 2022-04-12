# what3words (DNS edition)
Toy Go DNS server which resolves IPs based on a three.word.format

**Build the project locally:**
```bash
$ go build .
```

**Translate a domain to its three-word equivalent format:**
```bash
$ ./what3words -t bede.io
balcony.scissors.hurt
```

**Run a DNS server for the three-word format:**
```bash
$ sudo ./what3words -s
Listening on port 53...
```

**Using the DNS server (nslookup):**
```bash
$ nslookup balcony.scissors.hurt localhost
Server:		localhost
Address:	127.0.0.1#53

Name:	balcony.scissors.hurt
Address: 35.176.67.126
```

**Using the DNS server (dig):**
```bash
$ dig @localhost balcony.scissors.hurt

; <<>> DiG 9.10.6 <<>> @localhost balcony.scissors.hurt
; (2 servers found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 64864
;; flags: qr aa; QUERY: 0, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 0

;; ANSWER SECTION:
balcony.scissors.hurt.	0	IN	A	35.176.67.126

;; Query time: 0 msec
;; SERVER: 127.0.0.1#53(127.0.0.1)
;; WHEN: Tue Apr 12 17:55:28 BST 2022
;; MSG SIZE  rcvd: 49
```

## Using the DNS Server (Safari, ping, curl etc.)
MacOS doesn't seem to like my DNS server. I've tried a lot of things:
* Adding `127.0.0.1`, `0.0.0.0`, `localhost`, `mini.local` etc. to my DNS Servers/Domains preferences pane
* Using `networksetup` to set these programmatically instead of through the GUI
* Clearing the DNS cache using `sudo killall -HUP mDNSResponder`
* Restarting my computer (of course!)
* Explicitly specifying the `--dns-servers` option to curl
* Checking my `/etc/resolv.conf`
* Changing the order of the DNS servers in the prefpane
* Adding a known TLD like `.local` or `.com` to the end of the domains

But `ping` and `curl` resolutely report that they can't find the host I'm talking about, while refusing to make a DNS request to my server at all.
