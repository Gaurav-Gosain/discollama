package ollama

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

type APIClient struct {
	BaseURL string
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
	}
}

type Payload struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type TagModel struct {
	Details struct {
		Format            string      `json:"format"`
		Family            string      `json:"family"`
		Families          interface{} `json:"families"`
		ParameterSize     string      `json:"parameter_size"`
		QuantizationLevel string      `json:"quantization_level"`
	} `json:"details"`
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	Digest     string `json:"digest"`
	Size       int64  `json:"size"`
}

type TagResponse struct {
	Models []TagModel `json:"models"`
}

func (a *APIClient) OllamaModelNames() ([]string, error) {
	url := a.BaseURL + "/api/tags"

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(res.Body)
	var tags TagResponse
	err = decoder.Decode(&tags)
	if err != nil {
		return nil, err
	}

	if len(tags.Models) == 0 {
		return nil, errors.New("no models available")
	}

	modelNames := make([]string, 0, len(tags.Models))

	for _, model := range tags.Models {
		modelNames = append(modelNames, model.Name)
	}

	return modelNames, nil
}

func (client *APIClient) Generate(prompt, model string) (string, error) {
	url := client.BaseURL + "/api/generate"

	payload, err := json.Marshal(Payload{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(payload)))
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

type GenerateResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Context            []int  `json:"context"`
	TotalDuration      int64  `json:"total_duration"`
	LoadDuration       int64  `json:"load_duration"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalCount          int    `json:"eval_count"`
	EvalDuration       int64  `json:"eval_duration"`
	Done               bool   `json:"done"`
}
