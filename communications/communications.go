package communications

import (
	"fmt"
	"log/slog"

	"gorm.io/gorm"
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/responses"
)

type BroadcastType int

const (
	NewBet BroadcastType = iota
	StateUpdate
)

type Broadcast struct {
	Type BroadcastType
	Body interface{}
}

type ManagerEventType int

const (
	SubscribeFeed ManagerEventType = iota
	UnsubscribeFeed
	SubscribeChannel
	UnsubscribeChannel
	SubscribeAllBets
	UnsubscribeAllBets
	PropagateBet
	PropagateState
)

type ManagerEvent struct {
	Type ManagerEventType
	Body interface{}
}

type ManagerEventSubscribeFeed struct {
	Id   string
	Feed chan Broadcast
}
type ManagerEventUnsubscribeFeed struct {
	Id string
}

type ChannelType int

const (
	Bets ChannelType = iota
	ChatRoom
	Invoice
)

type ManagerEventSubscribeChannel struct {
	Id          string
	ChannelType ChannelType
	Channel     uint64
}
type ManagerEventUnsubscribeChannel struct {
	Id          string
	ChannelType ChannelType
	Channel     uint64
}
type ManagerEventSubscribeAllBets struct {
	Id string
}
type ManagerEventUnsubscribeAllBets struct {
	Id string
}

var ManagerPub *Manager

type Manager struct {
	Feeds             map[string]chan Broadcast
	SubscriptionsBets map[uint]map[string]bool
	ManagerReceiver   chan ManagerEvent
	Stop              chan bool
}

func (m *Manager) Run() {
	slog.Info("Starting manager")
	for true {
		select {
		case event := <-m.ManagerReceiver:
			slog.Info("Manager got event", "event", event)
			m.ProcessEvent(event)
		case <-m.Stop:
			slog.Info("Manager exiting")
			break
		}
	}
}

func New(Db *gorm.DB) *Manager {
	var games []db.Game
	err := Db.Find(&games).Error
	if err != nil {
		slog.Error("Error getting all the games", "err", err)
		panic("Error getting games")
	}

	subscriptions := make(map[uint]map[string]bool)
	for _, game := range games {
		subscriptions[game.ID] = make(map[string]bool)
	}

	ManagerPub = &Manager{
		Feeds:             make(map[string]chan Broadcast),
		SubscriptionsBets: subscriptions,
		ManagerReceiver:   make(chan ManagerEvent),
		Stop:              make(chan bool),
	}
	return ManagerPub
}

func (m *Manager) PropagateBet(bet responses.Bet) {
	subs, ok := m.SubscriptionsBets[bet.GameID]
	if !ok {
		slog.Error("Game not found", "game id", bet.GameID)
		return
	}
	for sub := range subs {
		feed, ok := m.Feeds[sub]
		if !ok {
			slog.Error("Feed not found", "sub", sub)
			continue
		}
		feed <- Broadcast{Type: NewBet, Body: bet}
	}
}

func (m *Manager) PropagateState(state db.GameState) {
	subs, ok := m.SubscriptionsBets[state.GameID]
	if !ok {
		slog.Error("Game not found", "game id", state.GameID)
		return
	}
	for sub := range subs {
		feed, ok := m.Feeds[sub]
		if !ok {
			slog.Error("Feed not found", "sub", sub)
			continue
		}
		feed <- Broadcast{Type: NewBet, Body: state}
	}
}

func (m *Manager) ProcessEvent(event ManagerEvent) {
	switch event.Type {
	case PropagateBet:
		bet, ok := event.Body.(responses.Bet)
		if !ok {
			panic(fmt.Sprintf("Cannot convert Bet %#v", event))
		}
		m.PropagateBet(bet)
		break
	case PropagateState:
		state, ok := event.Body.(db.GameState)
		if !ok {
			panic(fmt.Sprintf("Cannot convert GameState %#v", event))
		}
		m.PropagateState(state)
		break
	case SubscribeAllBets:
		sub, ok := event.Body.(ManagerEventSubscribeAllBets)
		if !ok {
			panic(fmt.Sprintf("Cannot convert ManagerEventSubscribeAllBets %#v", event))
		}

		for _, subs := range m.SubscriptionsBets {
			subs[sub.Id] = true
		}
		break
	case SubscribeChannel:
		sub, ok := event.Body.(ManagerEventSubscribeChannel)
		if !ok {
			panic(fmt.Sprintf("Cannot convert ManagerEventSubscribeChannel %#v", event))
		}
		switch sub.ChannelType {
		case Bets:
			m.SubscriptionsBets[uint(sub.Channel)][sub.Id] = true
			break
		case ChatRoom:
			break
		case Invoice:
			break
		default:
			panic(fmt.Sprintf("unexpected communications.ChannelType: %#v", sub.ChannelType))
		}
		break
	case SubscribeFeed:
		sub, ok := event.Body.(ManagerEventSubscribeFeed)
		if !ok {
			panic(fmt.Sprintf("Cannot convert SubscribeFeed %#v", event))
		}
		_, ok = m.Feeds[sub.Id]
		if ok {
			for _, subs := range m.SubscriptionsBets {
				delete(subs, sub.Id)
			}
		}
		m.Feeds[sub.Id] = sub.Feed
		break
	case UnsubscribeAllBets:
		sub, ok := event.Body.(ManagerEventUnsubscribeFeed)
		if !ok {
			panic(fmt.Sprintf("Cannot convert UnsubscribeFeed %#v", event))
		}
		for _, subs := range m.SubscriptionsBets {
			delete(subs, sub.Id)
		}
		delete(m.Feeds, sub.Id)
		break
	case UnsubscribeChannel:
	case UnsubscribeFeed:
	default:
		panic(fmt.Sprintf("unexpected communications.ManagerEventType: %#v", event.Type))
	}
}
