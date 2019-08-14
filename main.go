package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	li "logwrapper"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var config = Config{}
var client http.Client
var flFormatter string
var flLogLovel string
var flInterval int
var logger *li.StandardLogger

func init() {
	flag.StringVar(&flFormatter, "fmt", "json", "pick logger formatter: json(default) or text")
	flag.StringVar(&flLogLovel, "lvl", "info", "specify logging level, e.g. warn, info, debug")
	flag.IntVar(&flInterval, "interval", 24, "specify interface whithin we search for exceptions, default is 24")
	logger = li.NewLogger()
	config.Read()
}

type Timestamp time.Time

func (t *Timestamp) UnmarshalJSON(b []byte) error {
	s := string(b)
	ts, err := strconv.Atoi(strings.Split(s, ".")[0])
	if err != nil {
		return err
	}
	*t = Timestamp(time.Unix(int64(ts), 0))
	return nil
}

func (t *Timestamp) String() string {
	return time.Time(*t).String()
}

func (t *Timestamp) After(t1 time.Time) bool {
	return time.Time(*t).After(t1)
}

func (t *Timestamp) Before(t1 time.Time) bool {
	return time.Time(*t).Before(t1)
}

// Colletion of Message types.
type Messages []Message

// Colletion of Message types.
type ResponseData map[string]Message
type Data map[string]Message

func (d Data) GetBulk() [][]string {
	var list [][]string
	sortedByException := d.SetByException()
	index := 1
	for exception, messs := range sortedByException {
		for _, mess := range messs {
			var mode string
			if mess.Timestamp.Before(time.Now().Add(-24 * time.Hour)) {
				mode = "update"
			} else {
				mode = "dryrun"
			}
			list = append(list, []string{fmt.Sprintf("%d", index), mess.DeviceOs, exception, mess.DeivceName, mess.Timestamp.String(), mode})
			index++
		}
	}
	return list
}

func (d Data) SetByOS() map[string]Messages {
	set := map[string]Messages{}
	for _, v := range d {
		set[v.DeviceOs] = append(set[v.DeviceOs], v)
	}
	return set
}

func (d Data) SetByDC() map[string]Messages {
	set := map[string]Messages{}
	for _, v := range d {
		set[v.DeviceDC] = append(set[v.DeviceDC], v)
	}
	return set
}

func (d Data) SetByPlugin() map[string]Messages {
	set := map[string]Messages{}
	for _, v := range d {
		set[v.Name] = append(set[v.Name], v)
	}
	return set
}

func (d Data) SetByException() map[string]Messages {
	set := map[string]Messages{}
	for _, v := range d {
		set[v.Exception] = append(set[v.Exception], v)
	}
	return set
}

// A Message represents a celery message sent by python logging module.
// Logging is configred to send through HTTP POST only necessarily info.
type Message struct {
	Traceback  string     `json:"traceback"`
	Exception  string     `json:"exception"`
	Name       string     `json:"name"`
	Args       string     `json:"args"`
	Kwargs     string     `json:"kwargs"`
	State      string     `json:"state"`
	Started    *Timestamp `json:"started,omitempy"`
	Failed     *Timestamp `json:"failed,omitempy,"`
	DeivceName string     `json:"deivce_name"`
	DeviceOs   string     `json:"device_os"`
	DeviceArch string     `json:"device_arch"`
	DeviceSN   string     `json:"device_sn"`
	DeviceDC   string     `json:"devoce_dc"`
	Timestamp  *Timestamp `json:"timestamp,omitempty"` // equals to Failed timestamp
}

// Len, Swap and Less interfaces required to be implemented for each type which
// supossed to be sorted by sort.Sort func. From the doc:
// "Sort sorts data. It makes one call to data.Len to determine n, and O(n*log(n)) calls to data.Less and data.Swap.
func (messages Messages) Len() int {
	return len(messages)
}

func (messages Messages) Swap(i, j int) {
	messages[i], messages[j] = messages[j], messages[i]
}

func (messages Messages) Less(i, j int) bool {
	t2 := time.Time(*messages[j].Timestamp)
	return messages[i].Timestamp.Before(t2)
}

func (message Message) toString() string {
	var str strings.Builder
	str.WriteString(fmt.Sprintf("%s - %-47s - %s - %s - %s - %s\n",
		message.Timestamp.String(),
		message.Name,
		message.DeviceOs,
		message.DeivceName,
		message.DeviceDC,
		message.Exception))
	return str.String()
}

func handleFailedTasks() (error, bool) {
	now := time.Now()
	bod := now.Add(-time.Duration(flInterval) * time.Hour)
	resp, err := client.Get(fmt.Sprintf("%s?state=FAILURE&limit=0", config.FlowerApiUrl))
	if err != nil {
		return err, true
	}
	defer resp.Body.Close()

	var result ResponseData
	data := Data{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err, true
	}
	logger.Info("Got all failed tasks. Parsing...")
	var re = regexp.MustCompile(`(?m)(?P<field>'\w+')\: u(?P<value>'.*?')`)
	for k, v := range result {
		if !(v.Timestamp.After(bod) && v.Timestamp.Before(now)) {
			continue
		}
		for _, match := range re.FindAllStringSubmatch(v.Args, -1) {
			switch match[1] {
			case "'short_name'":
				v.DeivceName = match[2]
			case "'architecture'":
				v.DeviceArch = match[2]
			case "'os'":
				v.DeviceOs = match[2]
			case "'sn'":
				v.DeviceSN = match[2]
			case "'code'":
				v.DeviceDC = match[2]
			}
		}
		data[k] = v
	}
	subject := fmt.Sprint("Celery exceptions")
	receivers := config.Receivers
	r := NewRequest(receivers, subject)
	if err, ok := r.Send(data); !ok {
		return err, true
	}
	// for os, messs := range data.SetByOS() {
	// 	logger.Info(os)
	// 	for _, mess := range messs {
	// 		logger.Info(mess.toString())
	// 	}
	// }
	// for dc, messs := range data.SetByDC() {
	// 	logger.Info(dc)
	// 	for _, mess := range messs {
	// 		logger.Info(mess.toString())
	// 	}
	// }
	// for plugin, messs := range data.SetByPlugin() {
	// 	logger.Info(plugin)
	// 	for _, mess := range messs {
	// 		logger.Info(mess.toString())
	// 	}
	// }
	return nil, false
}

func main() {
	flag.Parse()
	if flFormatter == "text" {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}
	lvl, _ := logrus.ParseLevel(flLogLovel)
	logger.SetLevel(lvl)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client = http.Client{Transport: tr, Timeout: time.Second * 10}
	if err, ok := handleFailedTasks(); !ok {
		logger.Fatal(err)
	}
}
