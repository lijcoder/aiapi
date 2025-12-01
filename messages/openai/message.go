package openai

/*
OpenAI AI API
https://platform.openai.com/docs/api-reference/chat/create
*/

/* request params */
type Request struct {
	Model    string `json:"model,omitempty"`
	Messages []any  `json:"messages,omitempty"`
}

/* response params */
type Response struct {
}
