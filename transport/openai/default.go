package openai

import (
	"context"
	"fmt"
	"github.com/sashabaranov/go-openai"
)

type ChatGPT struct {
	client *openai.Client
}

func CreateBasePeerConnection(apiKey string) *ChatGPT {
	client := openai.NewClient(apiKey)

	return &ChatGPT{client}
}

func (this *ChatGPT) CreateChatCompletion() {
	resp, err := this.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Hello!",
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return
	}

	fmt.Println(resp.Choices[0].Message.Content)
}
