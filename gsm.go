package gogsmmodem

import (
	"bufio"
	"errors"
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/tarm/goserial"
)

type Modem struct {
	OOB   chan Packet
	Debug bool
	port  io.ReadWriteCloser
	rx    chan Packet
	tx    chan string
}

var OpenPort = func(config *serial.Config) (io.ReadWriteCloser, error) {
	return serial.OpenPort(config)
}

func Open(config *serial.Config, debug bool) (*Modem, error) {
	port, err := OpenPort(config)
	if debug {
		port = LogReadWriteCloser{port}
	}
	if err != nil {
		return nil, err
	}
	oob := make(chan Packet, 16)
	rx := make(chan Packet)
	tx := make(chan string)
	modem := &Modem{
		OOB:   oob,
		Debug: debug,
		port:  port,
		rx:    rx,
		tx:    tx,
	}
	// run send/receive goroutine
	go modem.listen()

	err = modem.init()
	if err != nil {
		return nil, err
	}
	return modem, nil
}

func (self *Modem) Close() error {
	close(self.OOB)
	close(self.rx)
	// close(self.tx)
	return self.port.Close()
}

// Commands

func (self *Modem) GetMessage(n int) (*Message, error) {
	packet, err := self.send("+CMGR", n)
	if err != nil {
		return nil, err
	}
	if msg, ok := packet.(Message); ok {
		return &msg, nil
	}
	return nil, errors.New("Message not found")
}

func (self *Modem) DeleteMessage(n int) error {
	_, err := self.send("+CMGD", n)
	return err
}

func lineChannel(r io.Reader) chan string {
	ret := make(chan string)
	go func() {
		buffer := bufio.NewReader(r)
		for {
			line, _ := buffer.ReadString(10)
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				continue
			}
			ret <- line
		}
	}()
	return ret
}

var reQuestion = regexp.MustCompile(`AT(\+[A-Z]+)`)

func parsePacket(status string, header string, body string) Packet {
	ls := strings.SplitN(header, ":", 2)
	if len(ls) != 2 {
		return UnknownPacket{header, []interface{}{}}
	}
	args := unquotes(strings.TrimSpace(ls[1]))
	switch ls[0] {
	case "+ZUSIMR":
		// message storage unset nag, ignore
		return nil
	case "+ZPASR":
		return ServiceStatus{args[0].(string)}
	case "+ZDONR":
		return NetworkStatus{args[0].(string)}
	case "+CMTI":
		return MessageNotification{args[0].(string), args[1].(int)}
	case "+CSCA":
		return SMSCAddress{args}
	case "+CMGR":
		return Message{Status: args[0].(string), Telephone: args[1].(string),
			Timestamp: parseTime(args[3].(string)), Body: body}
	case "":
		if status == "OK" {
			return OK{}
		} else {
			return ERROR{}
		}
	}
	return UnknownPacket{ls[0], args}
}

func (self *Modem) listen() {
	in := lineChannel(self.port)
	var echo, question, header, body string
	for {
		select {
		case line := <-in:
			if line == echo {
				continue // ignore echo of command
			} else if question != "" && strings.Index(line, question) == 0 {
				header = line
			} else if line == "OK" || line == "ERROR" {
				packet := parsePacket(line, header, body)
				self.rx <- packet
				header = ""
				body = ""
			} else if header != "" {
				// the body following a header
				body += line
			} else {
				// OOB packet
				p := parsePacket("OK", line, "")
				if p != nil {
					self.OOB <- p
				}
			}
		case line := <-self.tx:
			m := reQuestion.FindStringSubmatch(line)
			if len(m) > 0 {
				// command is a question
				question = m[1]
			}
			echo = strings.TrimRight(line, "\r\n")
			self.port.Write([]byte(line))
		}
	}
}

func formatCommand(cmd string, args ...interface{}) string {
	line := "AT" + cmd
	if len(args) > 0 {
		line += "=" + quotes(args)
	}
	line += "\r\n"
	return line
}

func (self *Modem) send(cmd string, args ...interface{}) (Packet, error) {
	self.tx <- formatCommand(cmd, args...)
	response := <-self.rx
	if _, e := response.(ERROR); e {
		return response, errors.New("Response was ERROR")
	}
	return response, nil
}

func (self *Modem) init() error {
	// clear settings
	if _, err := self.send("Z"); err != nil {
		return err
	}
	log.Println("Reset")
	// turn off echo
	if _, err := self.send("E0"); err != nil {
		return err
	}
	log.Println("Echo off")
	// set SMS storage
	// note: seems to deliver to SM (SIM) storage regardless, so need to set
	// READ to "SM" too.
	if _, err := self.send("+CPMS", "SM", "SM", "SM"); err != nil {
		return err
	}
	log.Println("Set SMS Storage")
	// set SMS text mode - easiest to implement. Ignore response which is
	// often a benign error.
	self.send("+CMGF", 1)

	log.Println("Set SMS text mode")
	// get SMSC
	// the modem complains if SMSC hasn't been set, but stores it correctly, so
	// query for stored value, then send a set from the query response.
	r, err := self.send("+CSCA?")
	if err != nil {
		return err
	}
	smsc := r.(SMSCAddress)
	log.Println("Got SMSC:", smsc.Args)
	r, err = self.send("+CSCA", smsc.Args...)
	if err != nil {
		return err
	}
	log.Println("Set SMSC to:", smsc.Args)
	// fmt.Println(r)
	return nil
}
