package api

import (
	"encoding/json"
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"greekkeepers.io/backend/auth"
	"greekkeepers.io/backend/communications"
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/engine"
	"greekkeepers.io/backend/requests"
	"greekkeepers.io/backend/responses"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func WebsocketsReader(conn *websocket.Conn, channel chan requests.WSrequest) {
	for {
		// Read message from client
		message := requests.WSrequest{}
		err := conn.ReadJSON(&message)
		if err != nil {
			slog.Error("Error while reading message", "err", err)
			break
		}
		channel <- message
	}

}
func WebsocketsHandler(c *gin.Context, sCtrl *SharedController) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("Upgrade failed", "err", err)
		return
	}
	readerChannel := make(chan requests.WSrequest)
	go WebsocketsReader(conn, readerChannel)

	managerFeed := make(chan communications.Broadcast)
	UUID := uuid.New()
	response := responses.WSresponse{
		Id:   0,
		Data: UUID,
	}
	conn.WriteJSON(&response)

	communications.ManagerPub.ManagerReceiver <- communications.ManagerEvent{
		Type: communications.SubscribeFeed,
		Body: communications.ManagerEventSubscribeFeed{
			Id:   UUID.String(),
			Feed: managerFeed,
		},
	}

	slog.Info("Connected", "conn", conn)

	userId := 0

	defer func() {
		conn.Close()
		communications.ManagerPub.ManagerReceiver <- communications.ManagerEvent{
			Type: communications.UnsubscribeFeed,
			Body: communications.ManagerEventUnsubscribeFeed{
				Id: UUID.String(),
			},
		}
	}()
	for {
		message := requests.WSrequest{}
		select {
		case response := <-managerFeed:
			conn.WriteJSON(response.Body)
			continue
		case recv := <-readerChannel:
			message = recv
		}

		response := responses.WSresponse{
			Id: message.Id,
		}
		switch message.Method {
		//case "ping":
		//	response.Data = "pong"
		//	conn.WriteJSON(&response)
		//	break
		case "auth":
			token := ""
			err := json.Unmarshal(message.Data, &token)
			if err != nil {
				slog.Error("Auth error", "err", err)
				return
			}
			claims, err := auth.VerifyToken(token, []byte(sCtrl.Env.PasswordSalt))
			if err != nil {
				slog.Error("Error verifying token", "err", err)
				return
			}

			sub, _ := claims.GetSubject()
			userid, err := strconv.Atoi(sub)
			userId = userid
			slog.Error("Auth successful", "userId", userId)
			break
		case "subscribe_bets":
			var games []uint64

			err := json.Unmarshal(message.Data, &games)
			if err != nil {
				slog.Error("Error subscribing to bets", "err", err)
				return
			}

			for _, game := range games {
				communications.ManagerPub.ManagerReceiver <- communications.ManagerEvent{
					Type: communications.SubscribeChannel,
					Body: communications.ManagerEventSubscribeChannel{
						Id:          UUID.String(),
						ChannelType: communications.Bets,
						Channel:     game,
					},
				}
			}
			break
		case "unsubscribe_bets":

			var games []uint64

			err := json.Unmarshal(message.Data, &games)
			if err != nil {
				slog.Error("Error unsubscribing to bets", "err", err)
				return
			}

			for _, game := range games {
				communications.ManagerPub.ManagerReceiver <- communications.ManagerEvent{
					Type: communications.UnsubscribeChannel,
					Body: communications.ManagerEventSubscribeChannel{
						Id:          UUID.String(),
						ChannelType: communications.Bets,
						Channel:     game,
					},
				}
			}
			break
		case "subscribe_all_bets":
			communications.ManagerPub.ManagerReceiver <- communications.ManagerEvent{
				Type: communications.SubscribeAllBets,
				Body: communications.ManagerEventSubscribeAllBets{
					Id: UUID.String(),
				},
			}
			break
		case "unsubscribe_all_bets":
			communications.ManagerPub.ManagerReceiver <- communications.ManagerEvent{
				Type: communications.UnsubscribeAllBets,
				Body: communications.ManagerEventUnsubscribeAllBets{
					Id: UUID.String(),
				},
			}
			break

		case "make_bet":
			if userId == 0 {
				continue
			}
			bet := requests.Bet{
				UserID: uint(userId),
				UUID:   UUID.String(),
			}
			err := json.Unmarshal(message.Data, &bet)
			if err != nil {
				slog.Error("Error making bet", "err", err)
				return
			}

			sCtrl.StatelessEngineChannel <- engine.Bet{
				IsContinue: false,
				Bet:        bet,
			}

			break

		case "continue_game":
			if userId == 0 {
				continue
			}
			bet := requests.ContinueGame{
				UserID: uint(userId),
				UUID:   UUID.String(),
			}
			err := json.Unmarshal(message.Data, &bet)
			if err != nil {
				slog.Error("Error continuing game", "err", err)
				return
			}
			sCtrl.StatelessEngineChannel <- engine.Bet{
				IsContinue: true,
				Bet:        bet,
			}

			break
		case "get_state":
			if userId == 0 {
				continue
			}
			req := requests.GetState{}
			err := json.Unmarshal(message.Data, &req)
			if err != nil {
				slog.Error("Error getting state", "err", err)
				return
			}
			var state db.GameState
			err = sCtrl.Db.Where("game_id=? AND user_id=? AND coin_id=?", req.GameID, userId, req.CoinID).First(&state).Error
			if err != nil {
				slog.Error("Error getting state", "err", err)
				return
			}
			response.Data = state
			conn.WriteJSON(&response)
			break
		case "get_uuid":
			response.Data = UUID
			conn.WriteJSON(&response)
			break

		default:
			break
		}
	}

}

func GameEndpoints(sCtrl *SharedController, router *gin.Engine) {
	router.GET("/game/ws", func(c *gin.Context) { WebsocketsHandler(c, sCtrl) })
}
