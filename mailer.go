// Mailer.

package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"gopkg.in/gomail.v2"
)

var tmpl *template.Template

type Request struct {
	from       string
	to         []string
	subject    string
	body       string
	attachment string
}

type Exceptions struct {
	Interval      int
	NewExceptions int
	Exceptions    [][]string
}

// Returns a pointer to the Request structure which represents just a collecton
// of email attributes.
func NewRequest(to []string, subject string) *Request {
	return &Request{
		to:      to,
		subject: subject,
	}
}

// Parses messages and creates an email body to be sent.
func (r *Request) parseMessages(d Data) error {
	var str strings.Builder
	var body bytes.Buffer
	tmpl, err := template.ParseFiles("templates/email.tmpl")
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	logger.Info("Parsing message")
	var exceptions Exceptions
	exceptions.Interval = flInterval
	exceptions.NewExceptions = len(d)
	exceptions.Exceptions = d.GetBulk()
	err = tmpl.Execute(&body, exceptions)
	str.WriteString(fmt.Sprintf("Exceptions raised by celery workers during last %d hours:\n\n", flInterval))
	str.WriteString(fmt.Sprintf("Number of new exceptions: %d", len(d)))
	str.WriteString("\n\n")
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "Gopher"
	}
	table := tablewriter.NewWriter(&str)
	table.SetHeader([]string{"Index", "OS", "Exception", "Device", "Date"})
	table.SetAutoMergeCells(true)
	table.SetRowLine(true)
	table.AppendBulk(d.GetBulk())
	table.Render()
	str.WriteString("\n\n")
	str.WriteString(fmt.Sprintf("This email is sent from: %s", hostname))
	r.body = body.String()
	logger.Debug(str.String())
	return nil
}

// Send an email. Please note, messages are sent without any encription.
func (r *Request) sendMail() bool {
	m := gomail.NewMessage()
	m.SetAddressHeader("From", config.Email, strings.ToUpper(strings.Split(config.Email, "@")[0]))
	for _, to := range r.to {
		m.SetAddressHeader("To", to, strings.ToUpper(strings.Split(to, ".")[0]))
	}
	m.SetHeader("Subject", r.subject)
	m.SetBody("text/html", r.body)
	d := gomail.Dialer{Host: config.Server, Port: config.Port}
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	if err := d.DialAndSend(m); err != nil {
		return false
	}
	return true
}

func (r *Request) Send(d Data) (error, bool) {
	err := r.parseMessages(d)
	if err != nil {
		return err, true
	}
	if flSendEmail {
		if ok := r.sendMail(); ok {
			logger.Infof("Email has been sent to %s\n", r.to)
		} else {
			logger.Infof("Failed to send the email to %s\n", r.to)
		}
	}
	return nil, false
}
