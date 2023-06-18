//go:generate mockery --output=../../mocks --name RabbitMQClient
package rmq

import (
	"context"
	"log"

	"github.com/wagslane/go-rabbitmq"
)

func NewConn(url string) (*rabbitmq.Conn, error) {
	return rabbitmq.NewConn(
		url,
		rabbitmq.WithConnectionOptionsLogging,
	)
}

func NewPublisher(
	conn *rabbitmq.Conn,
	exchangeName, exchangeType string, exchangeDeclare bool,
) (*rabbitmq.Publisher, error) {
	opt := []func(*rabbitmq.PublisherOptions){
		rabbitmq.WithPublisherOptionsLogging,
	}

	if exchangeName != "" {
		opt = append(opt, rabbitmq.WithPublisherOptionsExchangeName(exchangeName))
	}

	if exchangeType != "" {
		opt = append(opt, rabbitmq.WithPublisherOptionsExchangeKind(exchangeType))
	}

	if exchangeDeclare {
		opt = append(opt, rabbitmq.WithPublisherOptionsExchangeDeclare)
	}

	publisher, errNewPub := rabbitmq.NewPublisher(
		conn,
		opt...,
	)
	if errNewPub != nil {
		return nil, errNewPub
	}

	return publisher, nil
}

func PublishWithContext(
	ctx context.Context, conn *rabbitmq.Conn,
	routingKeysPub []string,
	contentType string,
	persistent bool, dataBytes []byte,
	correlationID, replyTo string,
	exchangeName, exchangeType string,
) error {
	publisher, errNewPub := NewPublisher(conn, exchangeName, exchangeType, false)
	if errNewPub != nil {
		return errNewPub
	}
	defer publisher.Close()

	publisher.NotifyPublish(func(c rabbitmq.Confirmation) {
		log.Printf("message confirmed from server. tag: %v, ack: %v", c.DeliveryTag, c.Ack)
	})

	opt := []func(*rabbitmq.PublishOptions){}

	if contentType != "" {
		opt = append(opt, rabbitmq.WithPublishOptionsContentType(contentType))
	}

	if persistent {
		opt = append(opt, rabbitmq.WithPublishOptionsPersistentDelivery)
	}

	if correlationID != "" {
		opt = append(opt, rabbitmq.WithPublishOptionsCorrelationID(correlationID))
	}

	if replyTo != "" {
		opt = append(opt, rabbitmq.WithPublishOptionsReplyTo(replyTo))
	}

	if exchangeName != "" {
		opt = append(opt, rabbitmq.WithPublishOptionsExchange(exchangeName))
	}

	errPub := publisher.PublishWithContext(
		ctx,
		dataBytes,
		routingKeysPub,
		opt...,
	)
	if errPub != nil {
		return errPub
	}

	return nil
}

func NewConsumer(
	conn *rabbitmq.Conn,
	handler rabbitmq.Handler,
	queueName, routingKeyCon string,
	queueDurable, autoAck bool,
	concurrency, prefetchCount int,
	exchangeName, exchangeType string, exchangeDeclare bool,
) (*rabbitmq.Consumer, error) {
	opt := []func(*rabbitmq.ConsumerOptions){
		rabbitmq.WithConsumerOptionsLogging,
	}

	if routingKeyCon != "" {
		opt = append(opt, rabbitmq.WithConsumerOptionsRoutingKey(routingKeyCon))
	}

	if queueDurable {
		opt = append(opt, rabbitmq.WithConsumerOptionsQueueDurable)
	}

	if autoAck {
		opt = append(opt, rabbitmq.WithConsumerOptionsConsumerAutoAck(autoAck))
	}

	if concurrency != 0 {
		opt = append(opt, rabbitmq.WithConsumerOptionsConcurrency(concurrency))
	}

	if prefetchCount != 0 {
		opt = append(opt, rabbitmq.WithConsumerOptionsQOSPrefetch(prefetchCount))
	}

	if exchangeName != "" {
		opt = append(opt, rabbitmq.WithConsumerOptionsExchangeName(exchangeName))
	}

	if exchangeType != "" {
		opt = append(opt, rabbitmq.WithConsumerOptionsExchangeKind(exchangeType))
	}

	if exchangeDeclare {
		opt = append(opt, rabbitmq.WithConsumerOptionsExchangeDeclare)
	}

	return rabbitmq.NewConsumer(
		conn,
		handler,
		queueName,
		opt...,
	)
}
