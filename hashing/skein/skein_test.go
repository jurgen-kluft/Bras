package skein

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
)

type katResult struct {
	stateSize     int
	hashBitLength int
	msgLength     int
	msg           []byte
	msgFill       int
	result        []byte
	resultFill    int
	macKeyLen     int
	macKey        []byte
	macKeyFill    int
	restOfLine    []byte
}

func TestSkein(t *testing.T) {
	scanner, e := newKatScanner("test_data/skein_golden_kat.txt")
	if e != nil {
		fmt.Printf("Error: %s\n", e)
		t.Error("Cannot open test vector file")
		return
	}
	if !checkKATVectors(scanner) {
		t.Fail()
	}

	skein, _ := New(256, 256)
	hash := make([]byte, 32)
	skein.UpdateBits([]byte{'C', 'a', 's', 'e'}, 4*8)
	skein.DoFinal(hash)
	fmt.Printf("Computed hash of 'Case':%s\n", hex.EncodeToString(hash))
}

func checkKATVectors(ks *katScanner) bool {
	kr := new(katResult)

	var tree, mac, normal int

	for ks.fillResult(kr) {
		if strings.Contains(string(kr.restOfLine), "Tree") {
			tree++
			continue
		}
		hash := make([]byte, (kr.hashBitLength+7)/8)
		if strings.Contains(string(kr.restOfLine), "MAC") {
			skein, _ := NewMac(kr.stateSize, kr.hashBitLength, kr.macKey)
			// do it twice time with same instance to check if context was reset correctly
			for i := 0; i < 2; i++ {
				skein.UpdateBits(kr.msg, kr.msgLength)
				skein.DoFinal(hash)
				if ret := bytes.Compare(hash, kr.result); ret != 0 {
					fmt.Printf("%d-%d-%d-%s\n", kr.stateSize, kr.hashBitLength, +kr.msgLength, string(kr.restOfLine))
					fmt.Printf("Computed mac:\n%s\n", hex.EncodeToString(hash))
					fmt.Printf("Expected mac:\n%s\n", hex.EncodeToString(kr.result))
					return false
				}
			}
			mac++
			continue
		}

		skein, _ := New(kr.stateSize, kr.hashBitLength)
		// do it twice time with same instance to check if context was reset correctly
		for i := 0; i < 2; i++ {
			skein.UpdateBits(kr.msg, kr.msgLength)
			skein.DoFinal(hash)
			if ret := bytes.Compare(hash, kr.result); ret != 0 {
				fmt.Printf("%d-%d-%d-%s\n", kr.stateSize, kr.hashBitLength, kr.msgLength, string(kr.restOfLine))
				fmt.Printf("Computed hash:\n%s\n", hex.EncodeToString(hash))
				fmt.Printf("Expected result:\n%s\n", hex.EncodeToString(kr.result))
				return false
			}
		}

		normal++
	}
	fmt.Printf("(Tree: %d), mac: %d, normal: %d, Skein tests total: %d\n", tree, mac, normal, mac+normal)
	return true
}

/*
 * Scanner for KAT file that fills in the KAT test vectors and
 * expected results.
 *
 */
const Start = 0
const MessageLine = 1
const Result = 2
const MacKeyHeader = 3
const MacKey = 4
const Done = 5

type katScanner struct {
	buf   *bufio.Reader
	state int
}

func newKatScanner(name string) (*katScanner, error) {
	r, e := os.Open(name)
	if e != nil {
		return nil, e
	}
	bufio := bufio.NewReader(r)
	return &katScanner{bufio, Start}, nil
}

/**
 * Fill in data from KAT file, one complete element at a time.
 *
 * @param kr The resulting KAT data
 * @return
 */
func (s *katScanner) fillResult(kr *katResult) bool {

	dataFound := false

	for s.state != Done {
		line, err := s.buf.ReadString('\n')
		if err != nil {
			break
		}
		s.parseLines(line, kr)
		dataFound = true
	}
	s.state = Start
	return dataFound
}

func (s *katScanner) parseLines(line string, kr *katResult) {
	//    fmt.Printf("Line: %s", line)

	line = strings.TrimSpace(line)

	if len(line) <= 1 {
		return
	}

	if strings.HasPrefix(line, "Message") {
		s.state = MessageLine
		return
	}
	if strings.HasPrefix(line, "Result") {
		s.state = Result
		return
	}
	if strings.HasPrefix(line, "MAC") {
		s.state = MacKeyHeader
	}
	if strings.HasPrefix(line, "------") {
		s.state = Done
		return
	}
	switch s.state {
	case Start:
		if strings.HasPrefix(line, ":Skein-") {
			s.parseHeaderLine(line, kr)
		} else {
			fmt.Printf("Wrong format found")
			os.Exit(1)
		}
	case MessageLine:
		s.parseMessageLine(line, kr)
	case Result:
		s.parseResultLine(line, kr)
	case MacKey:
		s.parseMacKeyLine(line, kr)
	case MacKeyHeader:
		s.parseMacKeyHeaderLine(line, kr)
	}
}

func (s *katScanner) parseHeaderLine(line string, kr *katResult) {
	var rest string

	ret, err := fmt.Sscanf(line, ":Skein-%d: %d-bit hash, msgLen = %d%s", &kr.stateSize, &kr.hashBitLength, &kr.msgLength, &rest)
	if err != nil {
		fmt.Printf("state size: %d, bit length: %d, msg length: %d, rest: %s, ret: %d\n", kr.stateSize, kr.hashBitLength, kr.msgLength, rest, ret)
	}

	idx := strings.Index(line, rest)
	kr.restOfLine = make([]byte, len(line)-idx)
	copy(kr.restOfLine[:], line[idx:])

	if kr.msgLength > 0 {
		if (kr.msgLength % 8) != 0 {
			kr.msg = make([]byte, (kr.msgLength>>3)+1)
		} else {
			kr.msg = make([]byte, kr.msgLength>>3)
		}
	}
	if (kr.hashBitLength % 8) != 0 {
		kr.result = make([]byte, (kr.hashBitLength>>3)+1)
	} else {
		kr.result = make([]byte, kr.hashBitLength>>3)
	}
	kr.msgFill = 0
	kr.resultFill = 0
	kr.macKeyFill = 0
}

func (s *katScanner) parseMessageLine(line string, kr *katResult) {
	var d [16]int

	if strings.Contains(line, "(none)") {
		return
	}
	ret, err := fmt.Sscanf(line, "%x%x%x%x%x%x%x%x%x%x%x%x%x%x%x%x", &d[0], &d[1], &d[2], &d[3], &d[4], &d[5], &d[6], &d[7], &d[8], &d[9], &d[10], &d[11], &d[12], &d[13], &d[14], &d[15])

	for i := 0; i < ret; i++ {
		kr.msg[kr.msgFill] = byte(d[i])
		kr.msgFill++
	}
	if err != nil && ret <= 0 {
		fmt.Printf("msg: %s, ret: %d, %s \n", hex.EncodeToString(kr.msg), ret, err)
	}
}

func (s *katScanner) parseResultLine(line string, kr *katResult) {
	var d [16]int

	ret, err := fmt.Sscanf(line, "%x%x%x%x%x%x%x%x%x%x%x%x%x%x%x%x", &d[0], &d[1], &d[2], &d[3], &d[4], &d[5], &d[6], &d[7], &d[8], &d[9], &d[10], &d[11], &d[12], &d[13], &d[14], &d[15])

	for i := 0; i < ret; i++ {
		kr.result[kr.resultFill] = byte(d[i])
		kr.resultFill++
	}
	if err != nil && ret <= 0 {
		fmt.Printf("result: %s, ret: %d, %s \n", hex.EncodeToString(kr.result))
	}
}

func (s *katScanner) parseMacKeyLine(line string, kr *katResult) {
	var d [16]int

	if strings.Contains(line, "(none)") {
		return
	}
	ret, err := fmt.Sscanf(line, "%x%x%x%x%x%x%x%x%x%x%x%x%x%x%x%x", &d[0], &d[1], &d[2], &d[3], &d[4], &d[5], &d[6], &d[7], &d[8], &d[9], &d[10], &d[11], &d[12], &d[13], &d[14], &d[15])

	for i := 0; i < ret; i++ {
		kr.macKey[kr.macKeyFill] = byte(d[i])
		kr.macKeyFill++
	}
	if err != nil && ret <= 0 {
		fmt.Printf("macKey: %s, ret: %d, %s \n", hex.EncodeToString(kr.macKey), ret, err)
	}
}

func (s *katScanner) parseMacKeyHeaderLine(line string, kr *katResult) {
	var rest string

	ret, err := fmt.Sscanf(line, "MAC key = %d%s", &kr.macKeyLen, &rest)

	if ret > 0 {
		kr.macKey = make([]byte, kr.macKeyLen)
	}
	if err != nil && ret <= 0 {
		fmt.Printf("macKeyLen: %d, %s\n", kr.macKeyLen, err)
	}
	s.state = MacKey
}
