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

type UserSettingHandler struct {
	rmqConn *rabbitmq.Conn
	logger  *zerolog.Logger
	svc     service.UserSettingServiceI
}

func NewUserSettingHandler(
	rmqConn *rabbitmq.Conn, logger *zerolog.Logger, svc service.UserSettingServiceI,
) UserSettingHandlerI {
	h := new(UserSettingHandler)
	h.rmqConn = rmqConn
	h.logger = logger
	h.svc = svc
	return h
}

func (h *UserSettingHandler) UpdateByUserID() (*rabbitmq.Consumer, error) {
	return rmq.NewConsumer(
		h.rmqConn,
		func(d rabbitmq.Delivery) rabbitmq.Action {
			log.Printf("consumed: %v", string(d.Body))

			setting := new(model.UserSetting)
			if errJSONUn := json.Unmarshal(d.Body, &setting); errJSONUn != nil {
				h.logger.Error().Err(errJSONUn).Msg("json unmarshal err")
				return rabbitmq.NackDiscard
			}

			ucErr := h.svc.UpdateByUserID(setting)
			if ucErr != nil {
				h.logger.Error().Err(ucErr).Msg("svc.UpdateByID err")
				return rabbitmq.NackDiscard
			}

			h.logger.Debug().Msgf("from %v, tag %v, success with id:%v", d.RoutingKey, d.DeliveryTag, setting.UserID)
			// rabbitmq.Ack, rabbitmq.NackDiscard, rabbitmq.NackRequeue
			return rabbitmq.Ack
		},
		"user_setting.updated", "user_setting.updated",
		true, false,
		0, 0,
		"user_setting", "topic", true,
	)

	// msgs, errSub := h.rmq.Subscribe(
	// 	"user_setting.updated",
	// 	"topic",
	// 	"user_setting.updated",
	// 	"user_setting.updated",
	// 	false,
	// 	1,
	// )
	// if errSub != nil {
	// 	h.logger.Error().Err(errSub).Msg("rmq.Subscribe err")
	// 	return
	// }

	// go func(deliveries <-chan amqp091.Delivery) {
	// 	newSetting := new(model.UserSetting)
	// 	for d := range deliveries {
	// 		_ = json.Unmarshal(d.Body, &newSetting)

	// 		// h.logger.Debug().Msgf("%v", newSetting)

	// 		setting, ucErr := h.svc.UpdateByUserID(newSetting)
	// 		if ucErr != nil {
	// 			h.logger.Error().Err(ucErr).Msgf("svc.UpdateByID err: %v\n%v", newSetting, setting)
	// 			// _ = d.Nack(false, false)
	// 			continue
	// 		}

	// 		_ = d.Ack(false)
	// 		h.logger.Debug().Msgf("from %s, rmq.Sub success with id:%v", d.RoutingKey, setting.ID)
	// 	}
	// }(msgs)
}
