package twitch

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/LinneB/twitchwh"
)

var TwitchwhClient *twitchwh.Client

func InitTwitchWhClient() (*twitchwh.Client, error) {
	client, err := twitchwh.New(twitchwh.ClientConfig{
		ClientID:      os.Getenv("TWITCH_CLIENT_ID"),
		ClientSecret:  os.Getenv("TWITCH_CLIENT_SECRET"),
		WebhookSecret: os.Getenv("EVENTSUB_SECRET"),
		WebhookURL:    os.Getenv("BOT_HOST") + "eventsub",
		Debug:         true,
	})
	if err != nil {
		return nil, err
	}
	TwitchwhClient = client
	return client, nil
}

// TODO: Finish eventsub handling
func InitTwitchEventSub() (func(w http.ResponseWriter, r *http.Request), error) {
	TwitchwhClient.RemoveSubscriptionByType("channel.channel_points_custom_reward_redemption.add", twitchwh.Condition{})
	TwitchwhClient.RemoveSubscriptionByType("channel.chat.message", twitchwh.Condition{})

	// Handle reward redemption events
	TwitchwhClient.On("channel.channel_points_custom_reward_redemption.add", func(event json.RawMessage) {
		var data RewardRedemptionEvent
		if err := json.Unmarshal(event, &data); err != nil {
			log.Printf("Error unmarshalling EventSub reward event: %v", err)
			return
		}
		HandleRewardRedemption(data.BroadcasterUserID, data.ID, data.Reward.ID, data.UserID, data.UserName, data.UserInput)
	})

	// Handle chat message events
	TwitchwhClient.On("channel.chat.message", func(event json.RawMessage) {
		var data ChatMessageEvent
		if err := json.Unmarshal(event, &data); err != nil {
			log.Printf("Error unmarshalling EventSub chat event: %v", err)
			return
		}
		HandleChatMessage(data.BroadcasterUserID, data.ChatterUserID, data.ChatterUserName, data.Message.Text)
	})

	return TwitchwhClient.Handler, nil
}

func AddStreamer(streamerId string) error {
	log.Println("Adding streamer to TwitchwhClient:", streamerId)

	// Subscribe to reward redemption events
	err := TwitchwhClient.AddSubscription("channel.channel_points_custom_reward_redemption.add", "1", twitchwh.Condition{
		BroadcasterUserID: streamerId,
	})
	if err != nil {
		log.Printf("Error subscribing to reward redemptions for %s: %v", streamerId, err)
		return err
	}

	// Subscribe to chat message events
	err = TwitchwhClient.AddSubscription("channel.chat.message", "1", twitchwh.Condition{
		BroadcasterUserID: streamerId,
		UserID:            streamerId, // Bot needs to be a moderator or the broadcaster
	})
	if err != nil {
		log.Printf("Error subscribing to chat messages for %s: %v", streamerId, err)
		return err
	}

	log.Printf("Successfully subscribed to events for streamer %s", streamerId)
	return nil
}
