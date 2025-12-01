package general

/*
general message api
role type: system、assistant、user
*/

/* request params */
type Request struct {
	Stream            bool              `json:"stream,omitempty"`
	Model             string            `json:"model,omitempty"`
	Contents          []Content         `json:"contents,omitempty"`
	Tools             []Tool            `json:"tools,omitempty"`
	SystemInstruction *Content          `json:"systemInstruction,omitempty"`
	GenerationConfig  *GenerationConfig `json:"generationConfig,omitempty"`
}

type Content struct {
	Role  string `json:"role,omitempty"`
	Parts []Part `json:"parts,omitempty"`
}

type Part struct {
	Text             *string           `json:"text,omitempty"`
	FunctionCall     *FunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
}

type FunctionCall struct {
	Id   string         `json:"id,omitempty"`
	Name string         `json:"name,omitempty"`
	Args map[string]any `json:"args,omitempty"`
}

type FunctionResponse struct {
	Id       string                  `json:"id,omitempty"`
	Name     string                  `json:"name,omitempty"`
	Response FunctionResponseContent `json:"response,omitempty"`
}

type FunctionResponseContent struct {
	Output *string `json:"output,omitempty"`
	Error  *string `json:"error,omitempty"`
}

type Tool struct {
	FunctionDeclarations []FunctionDeclaration `json:"functionDeclarations,omitempty"`
	WebSearch            *bool                 `json:"webSearch,omitempty"`
}

type FunctionDeclaration struct {
	Name        string                `json:"name,omitempty"`
	Description string                `json:"description,omitempty"`
	Parameters  *ParametersJsonSchema `json:"parameters,omitempty"`
}

type ParametersJsonSchema struct {
	Type       string         `json:"type,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
	Required   []string       `json:"required,omitempty"`
}

type GenerationConfig struct {
	StopSequences    []string `json:"stopSequences,omitempty"`
	MaxOutputTokens  *int     `json:"maxOutputTokens,omitempty"`
	Temperature      *float32 `json:"temperature,omitempty"`
	TopP             *float32 `json:"topP,omitempty"`
	TopK             *int     `json:"topK,omitempty"`
	PresencePenalty  *float32 `json:"presencePenalty,omitempty"`
	FrequencyPenalty *float32 `json:"frequencyPenalty,omitempty"`
	Logprobs         *int     `json:"logprobs,omitempty"`
}

/* response params */
