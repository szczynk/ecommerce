package handler

import (
	"encoding/json"
	"log"
	"user-consumer-go/model"
	"user-consumer-go/package/rmq"
	"user-consumer-go/service"

	"github.com/rs/zerolog"
	"github.com/wagslane/go-rabbitmq"
)

type UserHandler struct {
	rmqConn *rabbitmq.Conn
	logger  *zerolog.Logger
	svc     service.UserServiceI
}

func NewUserHandler(rmqConn *rabbitmq.Conn, logger *zerolog.Logger, svc service.UserServiceI) UserHandlerI {
	h := new(UserHandler)
	h.rmqConn = rmqConn
	h.logger = logger
	h.svc = svc
	return h
}

func (h *UserHandler) Create() (*rabbitmq.Consumer, error) {
	return rmq.NewConsumer(
		h.rmqConn,
		func(d rabbitmq.Delivery) rabbitmq.Action {
			log.Printf("consumed: %v", string(d.Body))

			newUser := new(model.User)
			if errJSONUn := json.Unmarshal(d.Body, &newUser); errJSONUn != nil {
				h.logger.Error().Err(errJSONUn).Msg("json unmarshal err")
				return rabbitmq.NackDiscard
			}

			ucErr := h.svc.Create(newUser)
			if ucErr != nil {
				h.logger.Error().Err(ucErr).Msg("svc.Create err")
				return rabbitmq.NackDiscard
			}

			h.logger.Debug().Msgf("from %v, tag %v, success with id:%v", d.RoutingKey, d.DeliveryTag, newUser.ID)
			// rabbitmq.Ack, rabbitmq.NackDiscard, rabbitmq.NackRequeue
			return rabbitmq.Ack
		},
		"user.created", "user.created",
		true, false,
		0, 0,
		"user", "topic", true,
	)
}

func (h *UserHandler) UpdateByID() (*rabbitmq.Consumer, error) {
	return rmq.NewConsumer(
		h.rmqConn,
		func(d rabbitmq.Delivery) rabbitmq.Action {
			log.Printf("consumed: %v", string(d.Body))

			user := new(model.User)
			if errJSONUn := json.Unmarshal(d.Body, &user); errJSONUn != nil {
				h.logger.Error().Err(errJSONUn).Msg("json unmarshal err")
				return rabbitmq.NackDiscard
			}

			ucErr := h.svc.UpdateByID(user)
			if ucErr != nil {
				h.logger.Error().Err(ucErr).Msg("svc.UpdateByID err")
				return rabbitmq.NackDiscard
			}

			h.logger.Debug().Msgf("from %v, tag %v, success with id:%v", d.RoutingKey, d.DeliveryTag, user.ID)
			// rabbitmq.Ack, rabbitmq.NackDiscard, rabbitmq.NackRequeue
			return rabbitmq.Ack
		},
		"user.updated", "user.updated",
		true, false,
		0, 0,
		"user", "topic", true,
	)
}
