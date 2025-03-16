package mail

import (
	"bytes"
	"errors"
	"janvlog/internal/libs/xerrors"
	"log/slog"
	"net/mail"
	"net/smtp"
	"strconv"
)

var ErrInvalidEmail = errors.New("invalid email")

type Config struct {
	Host     string
	Port     int
	From     string
	Username string
	Password string
}

func NewSender(cfg Config) *Sender {
	return &Sender{cfg: cfg}
}

type Sender struct {
	cfg Config
}

func (s *Sender) SendHTML(
	toEmails []string,
	title string,
	body []byte,
) error {
	pass, fail := splitBy(toEmails, valid)
	if len(pass) == 0 {
		return xerrors.Errorf("no valid emails: %v", fail)
	}

	slog.Info("failed mails: ", slog.Any("mails", fail))

	subjectfromto :=
		"Subject: " + title + "\n" +
			"From: " + s.cfg.From + "\n"
		// "To: " + toEmail + "\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

	var message bytes.Buffer

	message.WriteString(subjectfromto)
	message.WriteString(mime)
	message.Write(body)

	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)

	err := smtp.SendMail(
		s.cfg.Host+":"+strconv.Itoa(s.cfg.Port),
		auth,
		s.cfg.From,
		toEmails,
		message.Bytes(),
	)
	if err != nil {
		return xerrors.Wrap(err, "smtp.SendMail")
	}

	return nil
}

func valid(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func splitBy[T any](arr []T, splitter func(T) bool) ([]T, []T) {
	var pass, fail []T

	for _, v := range arr {
		if splitter(v) {
			pass = append(pass, v)
		} else {
			fail = append(fail, v)
		}
	}

	return pass, fail
}
