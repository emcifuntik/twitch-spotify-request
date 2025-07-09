package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/nicklaw5/helix/v2"
)

func main() {
	// Create a client with HTTP/1.1 forced for POST requests
	httpClient := &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: false, // Force HTTP/1.1
			TLSNextProto:      make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		},
	}

	client, err := helix.NewClient(&helix.Options{
		ClientID:        "3jtnnvm6teiohy6cl6678hjqdlycao",
		ClientSecret:    "wiyyvhr1gaegy53swqt25wmrdpuvxq",
		UserAccessToken: "5cl4sqr4y3miosegpq6354la6xfag6",
		RefreshToken:    "bpfk7gxi5v7r9goblz6mwt30xkbjsezxrylnxh6oc5f0pijmrk",
		HTTPClient:      httpClient, // Use HTTP/1.1 client
	})

	if err != nil {
		panic(err)
	}

	// Test with a simple GET request first
	resp, err := client.GetUsers(&helix.UsersParams{
		IDs: []string{"129095615"},
	})

	if err != nil {
		panic(err)
	}

	fmt.Printf("User request successful: %+v\n", resp.Data.Users)

	// // Now try the custom reward creation with different parameters
	// rewardResp, err := client.CreateCustomReward(&helix.ChannelCustomRewardsParams{
	// 	BroadcasterID: "129095615",
	// 	Title:         "Working Reward " + fmt.Sprintf("%d", time.Now().Unix()),
	// 	Cost:          999,
	// })

	// if err != nil {
	// 	fmt.Printf("Error creating custom reward: %v\n", err)
	// 	panic(err)
	// }

	// fmt.Printf("Custom reward created successfully! ID: %s\n", rewardResp.Data.ChannelCustomRewards[0].ID)

	// Test SendChatMessage
	fmt.Println("Testing SendChatMessage...")
	chatResp, err := client.SendChatMessage(&helix.SendChatMessageParams{
		BroadcasterID: "129095615",
		SenderID:      "129095615", // Send as the broadcaster
		Message:       "Test message from Helix API - " + fmt.Sprintf("%d", time.Now().Unix()),
	})

	if err != nil {
		fmt.Printf("Error sending chat message: %v\n", err)
	} else {
		fmt.Printf("Chat message sent successfully! Response: %+v\n", chatResp)
		if chatResp.Error != "" {
			fmt.Printf("Helix API error: %s - %s\n", chatResp.Error, chatResp.ErrorMessage)
		}
	}
}
