package mail

import (
	"bytes"
	"errors"
	"janvlog/internal/libs/xerrors"
	"log/slog"
	"net/mail"
	"net/smtp"
	"strconv"

	"github.com/domodwyer/mailyak/v3"
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
	html []byte,
	csv []byte,
) error {
	pass, fail := splitBy(toEmails, valid)
	if len(pass) == 0 {
		return xerrors.Errorf("no valid emails: %v", fail)
	}

	slog.Info("failed mails: ", slog.Any("mails", fail))

	mail := mailyak.New(
		s.cfg.Host+":"+strconv.Itoa(s.cfg.Port),
		smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host),
	)

	mail.To(pass...)
	mail.From(s.cfg.From)
	mail.Subject(title)
	mail.HTML().Set(string(html))
	mail.Attach("report.csv", bytes.NewReader(csv))

	if err := mail.Send(); err != nil {
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
