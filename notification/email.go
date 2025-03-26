package notification

import (
	"bytes"
	"encoding/json"
	"fhir-alarm/config"
	"github.com/alecthomas/chroma/v2/quick"
	"github.com/wneessen/go-mail"
	"log/slog"
	"strings"
)

type EmailClient struct {
	Sender     string
	Recipients []string
	client     *mail.Client
}

func NewEmailClient(config config.Email) *EmailClient {
	client, err := mail.NewClient(config.Smtp.Server,
		mail.WithSMTPAuth(mail.SMTPAuthLogin), mail.WithTLSPortPolicy(mail.TLSMandatory),
		mail.WithUsername(config.Smtp.User), mail.WithPassword(config.Smtp.Password),
	)
	if err != nil {
		slog.Error("Failed to create e-mail client", "error", err)
		return nil
	}

	return &EmailClient{
		Sender:     config.Sender,
		Recipients: strings.Split(config.Recipients, ","),
		client:     client,
	}
}

func (c *EmailClient) Send(msg []byte) {
	message := mail.NewMsg()
	if err := message.From(c.Sender); err != nil {
		slog.Error("Failed to set FROM address.", "sender", c.Sender, "error", err)
	}
	if err := message.To(c.Recipients...); err != nil {
		slog.Error("Failed to set TO address.", "recipients", c.Recipients, "error", err)
	}
	message.Subject("💣 DSF Task failed")

	var msgf bytes.Buffer
	if err := json.Indent(&msgf, msg, "", "  "); err != nil {
		slog.Error("Failed to parse JSON", "error", err)
		return
	}
	var out bytes.Buffer
	if err := quick.Highlight(&out, msgf.String(), "json", "html", "monokailight"); err != nil {
		slog.Error("Failed to highlight JSON", "error", err)
		return
	}

	message.SetBodyString(mail.TypeTextHTML, out.String())

	if err := c.client.DialAndSend(message); err != nil {
		slog.Error("Failed to deliver E-Mail", "error", err)
		return
	}
	slog.Info("E-Mail notification successfully delivered")
}
