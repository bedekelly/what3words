package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

func realDomainToTriple(domain string, words []string) string {
	ips, err := net.LookupIP(domain)
	if err != nil {
		panic(err)
	}

	var ip net.IP
	for _, newIP := range ips {
		if newIP.To4() != nil {
			ip = newIP.To4()
			break
		}
	}

	if ip == nil {
		fmt.Println(fmt.Errorf("couldn't find IPv4 address for domain: %s", domain))
		os.Exit(1)
	}

	fourByte := ip.To4()
	ipNumber := binary.BigEndian.Uint32(fourByte)

	firstIndex := ipNumber >> 22 & 0x7FF
	secondIndex := ipNumber >> 11 & 0x7FF
	thirdIndex := ipNumber & 0x7FF

	return fmt.Sprintf("%s.%s.%s", words[firstIndex], words[secondIndex], words[thirdIndex])
}

func bytesToDomainSlice(queriesSection []byte) []string {
	domainPieces := []string{}

	r := bytes.NewReader(queriesSection)

	for {
		b, err := r.ReadByte()
		if err != nil || b == 0x0 {
			break
		}

		piece := make([]byte, int(b))
		r.Read(piece)
		domainPieces = append(domainPieces, string(piece))
	}

	return domainPieces
}

func domainSliceToBytes(pieces []string) []byte {
	bytes := make([]byte, 0, 256)
	for _, piece := range pieces {
		length := len(piece)
		bytes = append(bytes, byte(length))
		bytes = append(bytes, []byte(piece)...)
	}
	bytes = append(bytes, 0)
	return bytes
}

func getResponseHeaders(dataSlice []byte) []byte {
	// Set the query/response bit to RESPONSE.
	var QR byte = 0b1 << 7

	// Responding to a standard query only.
	var OPCode byte = 0b0 << 3

	// We're an authority on these triple-names.
	var AA byte = 0b1 << 2

	// We'll never truncate a message.
	var TC byte = 0b0 << 1

	// Recursion desired? (Copy from request.)
	var RD byte = dataSlice[3] & 0x2

	// We can't provide recursion.
	var RA byte = 0b0 << 7

	// Z must always be zeroes.
	var Z byte = 0x0

	// No error from us!
	var RCode byte = 0x0

	return []byte{
		// ID (2 bytes)
		dataSlice[0], dataSlice[1],

		// Flags (1 byte)
		QR | OPCode | AA | TC | RD,

		// More flags (1 byte)
		RA | Z | RCode,

		// Number of entries in the question section
		0, 0,

		// Number of resource records in the answer section
		0, 1,

		// Number of NS resource records in the answer section
		0, 0,

		// Number of resource records in the additional records section
		0, 0,
	}
}

func getResponseBody(ipNum uint32) []byte {
	ip1 := byte(ipNum >> 24 & 0xFF)
	ip2 := byte(ipNum >> 16 & 0xFF)
	ip3 := byte(ipNum >> 8 & 0xFF)
	ip4 := byte(ipNum >> 0 & 0xFF)

	fmt.Printf("Responding with IP: %d.%d.%d.%d\n", ip1, ip2, ip3, ip4)

	return []byte{
		// TYPE = A Record
		0, 1,

		// CLASS = Internet
		0, 1,

		// TTL = 0 to prevent any caching
		0, 0, 0, 0,

		// RDLength = 4, a standard 4-byte IPv4 address.
		0, 4,

		// IP address as requested.
		ip1, ip2, ip3, ip4,
	}
}

func processDNSRequest(dataSlice []byte, wordIndices map[string]int, socket *net.UDPConn, remoteAddr *net.UDPAddr) {
	transactionID := dataSlice[0:2]
	fmt.Printf("Transaction ID: 0x%X\n", transactionID)

	flags := dataSlice[2:4]
	fmt.Printf("Flags: 0x%X\n", flags)

	questions := dataSlice[4:6]
	numQuestions := binary.BigEndian.Uint16(questions)
	fmt.Printf("Questions: %x\n", numQuestions)

	if numQuestions != 1 {
		fmt.Printf("Too many questions; not responding.")
		return
	}

	queries := dataSlice[12:]
	name := bytesToDomainSlice(queries)
	fmt.Printf("Name: %s\n", strings.Join(name, "."))

	// Build up the IP number from the given name, 11 bits at a time.
	var ipNum uint32 = 0
	allPresent := true
	for _, piece := range name {
		val, present := wordIndices[piece]
		if !present {
			allPresent = false
			break
		}
		ipNum = (ipNum << 11) | uint32(val)
	}

	if !allPresent {
		fmt.Println("Not all words were found in wordlist.")
		return
	}

	// Build up response from headers, domain and body.
	response := getResponseHeaders(dataSlice)
	domainNameAsBytes := domainSliceToBytes(name)
	response = append(response, domainNameAsBytes...)
	body := getResponseBody(ipNum)
	response = append(response, body...)

	socket.WriteToUDP(
		response,
		remoteAddr,
	)
}

func serveWordsDNS(wordIndices map[string]int) {
	socket, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 53,
	})

	if err != nil {
		panic("Listening on port 53 failed.")
	}

	fmt.Println("Listening on port 53...")

	defer socket.Close()

	for {
		requestData := make([]byte, 1024)
		readn, remoteAddr, err := socket.ReadFromUDP(requestData)

		if err != nil {
			panic(err)
		}

		go processDNSRequest(requestData[:readn], wordIndices, socket, remoteAddr)
	}
}

func main() {
	// Parse command-line arguments
	domainToTranslate := flag.String("t", "", "A domain to translate to a triple-word representation.")
	shouldServeDNS := flag.Bool("s", false, "Start a DNS server on the default port.")
	flag.Parse()

	// Load the wordlist from disk.
	wordsData, err := os.ReadFile("wordlist.txt")
	if err != nil {
		panic(err)
	}
	wordsStr := string(wordsData)
	words := strings.Split(wordsStr, "\n")

	// Translate from one domain format to another.
	if *domainToTranslate != "" {
		fmt.Println(realDomainToTriple(*domainToTranslate, words))
		os.Exit(0)
	}

	// Respond to DNS queries for this name type.
	if *shouldServeDNS {
		// Create a map so we don't have to loop through the list of words each time.
		wordIndices := make(map[string]int)
		for n, word := range words {
			wordIndices[word] = n
		}
		serveWordsDNS(wordIndices)
	}
}
