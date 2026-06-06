package clientapp

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (c *ChatClient) NewChatWindow() {
	c.chatWindow = c.app.NewWindow("BroChat")

	// Container for the chat window
	c.chatHistory = container.NewVBox()
	c.chatHistoryScroll = container.NewScroll(c.chatHistory)

	// Input for the user
	input := widget.NewEntry()
	input.PlaceHolder = "Message"
	input.OnSubmitted = func(text string) {
		if text == "" {
			return
		}

		input.SetText("")
		c.chatHistory.Add(widget.NewLabel("Me: " + text))
		c.chatHistoryScroll.ScrollToBottom()

		go c.SendMessage(text)
	}

	c.chatWindow.SetContent(container.NewBorder(nil, input, nil, nil, c.chatHistoryScroll))
	c.chatWindow.Resize(fyne.NewSize(400, 600))
}

func (c *ChatClient) addMessageToChat(message string) {
	newLabel := widget.NewLabel(message)
	c.chatHistory.Add(newLabel)
	c.chatHistoryScroll.ScrollToBottom()
}
