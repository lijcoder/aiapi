package gemini

/*
Gemini AI API
https://ai.google.dev/api/generate-content#v1beta.Candidate
*/

type GeminiContext struct {
	Stream          bool       `json:"stream,omitempty"`
	Model           string     `json:"model,omitempty"`
	Request         *Request   `json:"request,omitempty"`
	Response        *Response  `json:"response,omitempty"`
	StreamResponses []Response `json:"streamResponses,omitempty"`
}

/* request param */
type Request struct {
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
	GoogleSearch         *map[string]any       `json:"googleSearch,omitempty"`
}

type FunctionDeclaration struct {
	Name                 string                `json:"name,omitempty"`
	Description          string                `json:"description,omitempty"`
	Parameters           *map[string]any       `json:"parameters,omitempty"`
	ParametersJsonSchema *ParametersJsonSchema `json:"parametersJsonSchema,omitempty"`
}

type ParametersJsonSchema struct {
	Type       string         `json:"type,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
	Required   []string       `json:"required,omitempty"`
}

type GenerationConfig struct {
	StopSequences    []string        `json:"stopSequences,omitempty"`
	MaxOutputTokens  *int            `json:"maxOutputTokens,omitempty"`
	Temperature      *float32        `json:"temperature,omitempty"`
	TopP             *float32        `json:"topP,omitempty"`
	TopK             *int            `json:"topK,omitempty"`
	PresencePenalty  *float32        `json:"presencePenalty,omitempty"`
	FrequencyPenalty *float32        `json:"frequencyPenalty,omitempty"`
	Logprobs         *int            `json:"logprobs,omitempty"`
	ThinkingConfig   *ThinkingConfig `json:"thinkingConfig,omitempty"`
}

type ThinkingConfig struct {
	IncludeThoughts *bool   `json:"includeThoughts,omitempty"`
	ThinkingBudget  *int    `json:"thinkingBudget,omitempty"`
	ThinkingLevel   *string `json:"thinkingLevel,omitempty"`
}

/* response params */

type Response struct {
	Candidates    []Candidate    `json:"candidates,omitempty"`
	UsageMetadata *UsageMetadata `json:"usageMetadata,omitempty"`
	ResponseId    *string        `json:"responseId,omitempty"`
	ModelVersion  *string        `json:"modelVersion,omitempty"`
}

type UsageMetadata struct {
	PromptTokenCount           *int                 `json:"promptTokenCount,omitempty"`
	CachedContentTokenCount    *int                 `json:"cachedContentTokenCount,omitempty"`
	CandidatesTokenCount       *int                 `json:"candidatesTokenCount,omitempty"`
	ToolUsePromptTokenCount    *int                 `json:"toolUsePromptTokenCount,omitempty"`
	ThoughtsTokenCount         *int                 `json:"thoughtsTokenCount,omitempty"`
	TotalTokenCount            *int                 `json:"totalTokenCount,omitempty"`
	PromptTokensDetails        []ModalityTokenCount `json:"promptTokensDetails,omitempty"`
	CacheTokensDetails         []ModalityTokenCount `json:"cacheTokensDetails,omitempty"`
	CandidatesTokensDetails    []ModalityTokenCount `json:"candidatesTokensDetails,omitempty"`
	ToolUsePromptTokensDetails []ModalityTokenCount `json:"toolUsePromptTokensDetails,omitempty"`
}

type ModalityTokenCount struct {
	Modality   *string `json:"modality,omitempty"`
	TokenCount *int    `json:"tokenCount,omitempty"`
}

type Candidate struct {
	Content          *Content          `json:"content,omitempty"`
	FinishReason     *string           `json:"finishReason,omitempty"`
	CitationMetadata *CitationMetadata `json:"citationMetadata,omitempty"`
	TokenCount       *int              `json:"tokenCount,omitempty"`
	Index            *int              `json:"index,omitempty"`
	FinishMessage    *string           `json:"finishMessage,omitempty"`
	AvgLogprobs      *float32          `json:"avgLogprobs,omitempty"`
}

type CitationMetadata struct {
	CitationSources []CitationSource `json:"citationSources,omitempty"`
}

type CitationSource struct {
	StartIndex *int    `json:"startIndex,omitempty"`
	EndIndex   *int    `json:"endIndex,omitempty"`
	Uri        *string `json:"uri,omitempty"`
	License    *string `json:"license,omitempty"`
}
