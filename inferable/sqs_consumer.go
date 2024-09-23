package inferable

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// MessageHandler is a function type that processes SQS messages
type MessageHandler func(msg *sqs.Message) error

// SQSConsumer represents an SQS consumer
type SQSConsumer struct {
	svc            *sqs.SQS
	queueURL       string
	handler        MessageHandler
	pollInterval   time.Duration
	maxMessages    int64
	visibleTimeout int64
}

// NewSQSConsumer creates a new SQS consumer
func NewSQSConsumer(region, queueURL string, handler MessageHandler, accessKeyID, secretAccessKey, sessionToken string) (*SQSConsumer, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, err
	}

	return &SQSConsumer{
		svc:            sqs.New(sess),
		queueURL:       queueURL,
		handler:        handler,
		pollInterval:   20 * time.Second, // Default to long polling
		maxMessages:    10,               // Default to 10 messages per batch
		visibleTimeout: 30,               // Default visibility timeout of 30 seconds
	}, nil
}

// Start begins polling for messages
func (c *SQSConsumer) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			err := c.poll(ctx)
			if err != nil {
				return err
			}
		}

		time.Sleep(c.pollInterval)
	}
}

func (c *SQSConsumer) poll(ctx context.Context) error {
	output, err := c.svc.ReceiveMessageWithContext(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(c.queueURL),
		MaxNumberOfMessages: aws.Int64(c.maxMessages),
		VisibilityTimeout:   aws.Int64(c.visibleTimeout),
		WaitTimeSeconds:     aws.Int64(20), // Enable long polling
	})

	if err != nil {
		log.Printf("Error receiving SQS message: %v", err)
		return err
	}

	for _, message := range output.Messages {
		if err := c.handler(message); err != nil {
			log.Printf("Error processing message: %v", err)
			continue
		}

		_, err := c.svc.DeleteMessage(&sqs.DeleteMessageInput{
			QueueUrl:      aws.String(c.queueURL),
			ReceiptHandle: message.ReceiptHandle,
		})

		if err != nil {
			log.Printf("Error deleting message: %v", err)
		}
	}

	return nil
}

// SetPollInterval sets the polling interval
func (c *SQSConsumer) SetPollInterval(d time.Duration) {
	c.pollInterval = d
}

// SetMaxMessages sets the maximum number of messages to receive in one batch
func (c *SQSConsumer) SetMaxMessages(n int64) {
	c.maxMessages = n
}

// SetVisibilityTimeout sets the visibility timeout for received messages
func (c *SQSConsumer) SetVisibilityTimeout(seconds int64) {
	c.visibleTimeout = seconds
}
