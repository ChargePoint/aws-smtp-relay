package relay

import (
	"context"
	"net"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/pinpointemail"
	"github.com/aws/aws-sdk-go-v2/service/pinpointemail/types"
	"github.com/blueimp/aws-smtp-relay/internal/relay"
)

type pinpointemailSendEmailAPI interface {
	SendEmail(ctx context.Context, params *pinpointemail.SendEmailInput, optFns ...func(*pinpointemail.Options)) (*pinpointemail.SendEmailOutput, error)
}

// Client implements the Relay interface.
type Client struct {
	pinpointAPI     pinpointemailSendEmailAPI
	setName         *string
	allowFromRegExp *regexp.Regexp
	denyToRegExp    *regexp.Regexp
}

// Send uses the given Pinpoint API to send email data
func (c Client) Send(
	origin net.Addr,
	from string,
	to []string,
	data []byte,
) error {
	allowedRecipients, deniedRecipients, err := relay.FilterAddresses(
		from,
		to,
		c.allowFromRegExp,
		c.denyToRegExp,
	)
	if err != nil {
		relay.Log(origin, &from, deniedRecipients, err)
	}
	if len(allowedRecipients) > 0 {
		_, err := c.pinpointAPI.SendEmail(context.TODO(), &pinpointemail.SendEmailInput{
			ConfigurationSetName: c.setName,
			FromEmailAddress:     &from,
			Destination: &types.Destination{
				ToAddresses: allowedRecipients,
			},
			Content: &types.EmailContent{
				Raw: &types.RawMessage{
					Data: data,
				},
			},
		})
		relay.Log(origin, &from, allowedRecipients, err)
		if err != nil {
			return err
		}
	}
	return err
}

// New creates a new client with a session.
func New(
	configurationSetName *string,
	allowFromRegExp *regexp.Regexp,
	denyToRegExp *regexp.Regexp,
) Client {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic(err)
	}

	return Client{
		pinpointAPI:     pinpointemail.NewFromConfig(cfg),
		setName:         configurationSetName,
		allowFromRegExp: allowFromRegExp,
		denyToRegExp:    denyToRegExp,
	}
}
