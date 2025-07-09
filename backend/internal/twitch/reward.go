package twitch

import "time"

type RewardRedemptionEvent struct {
	ID                   string    `json:"id"`
	BroadcasterUserID    string    `json:"broadcaster_user_id"`
	BroadcasterUserLogin string    `json:"broadcaster_user_login"`
	BroadcasterUserName  string    `json:"broadcaster_user_name"`
	UserID               string    `json:"user_id"`
	UserLogin            string    `json:"user_login"`
	UserName             string    `json:"user_name"`
	UserInput            string    `json:"user_input"`
	Status               string    `json:"status"`
	Reward               Reward    `json:"reward"`
	RedeemedAt           time.Time `json:"redeemed_at"`
}

type Reward struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Cost   int    `json:"cost"`
	Prompt string `json:"prompt"`
}

type ChatMessageEvent struct {
	BroadcasterUserID           string  `json:"broadcaster_user_id"`
	BroadcasterUserLogin        string  `json:"broadcaster_user_login"`
	BroadcasterUserName         string  `json:"broadcaster_user_name"`
	ChatterUserID               string  `json:"chatter_user_id"`
	ChatterUserLogin            string  `json:"chatter_user_login"`
	ChatterUserName             string  `json:"chatter_user_name"`
	MessageID                   string  `json:"message_id"`
	Message                     Message `json:"message"`
	Color                       string  `json:"color"`
	Badges                      []Badge `json:"badges"`
	MessageType                 string  `json:"message_type"`
	Cheer                       *Cheer  `json:"cheer,omitempty"`
	Reply                       *Reply  `json:"reply,omitempty"`
	ChannelPointsCustomRewardID *string `json:"channel_points_custom_reward_id,omitempty"`
}

type Message struct {
	Text      string     `json:"text"`
	Fragments []Fragment `json:"fragments"`
}

type Fragment struct {
	Type      string     `json:"type"`
	Text      string     `json:"text"`
	Cheermote *Cheermote `json:"cheermote,omitempty"`
	Emote     *Emote     `json:"emote,omitempty"`
	Mention   *Mention   `json:"mention,omitempty"`
}

type Badge struct {
	SetID string `json:"set_id"`
	ID    string `json:"id"`
	Info  string `json:"info"`
}

type Cheermote struct {
	Prefix string `json:"prefix"`
	Bits   int    `json:"bits"`
	Tier   int    `json:"tier"`
}

type Emote struct {
	ID         string `json:"id"`
	EmoteSetID string `json:"emote_set_id"`
}

type Mention struct {
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	UserLogin string `json:"user_login"`
}

type Cheer struct {
	Bits int `json:"bits"`
}

type Reply struct {
	ParentMessageID   string `json:"parent_message_id"`
	ParentMessageBody string `json:"parent_message_body"`
	ParentUserID      string `json:"parent_user_id"`
	ParentUserName    string `json:"parent_user_name"`
	ParentUserLogin   string `json:"parent_user_login"`
	ThreadMessageID   string `json:"thread_message_id"`
	ThreadUserID      string `json:"thread_user_id"`
	ThreadUserName    string `json:"thread_user_name"`
	ThreadUserLogin   string `json:"thread_user_login"`
}
