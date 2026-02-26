package models

import (
	commontypes "owl/common_types"
	"owl/data"
	claude_model "owl/models/claude"
	gemeni_model "owl/models/gemeni"
	grok_model "owl/models/grok"
	ollama_model "owl/models/ollama"
	openai_4o_model "owl/models/open-ai-4o"
	openai_base "owl/models/open-ai-base"
	open_ai_gpt_model "owl/models/open-ai-gpt"
)

func GetModelForQuery(
	requestedModel string,
	context *data.Context,
	responseHandler commontypes.ResponseHandler,
	historyRepository data.HistoryRepository,
	streamMode bool,
	thinkingMode bool,
	streamThinkingMode bool,
	outputThinkingMode bool,
) (commontypes.Model, string) {

	modelToUse := requestedModel
	if modelToUse == "" && context != nil && context.PreferredModel != "" {
		modelToUse = context.PreferredModel
	}
	if modelToUse == "" {
		modelToUse = "claude"
	}

	var model commontypes.Model

	switch modelToUse {
	case "gemeni":
		model = &gemeni_model.GemeniModel{
			OpenAICompatibleModel: openai_base.OpenAICompatibleModel{
				ResponseHandler:   responseHandler,
				HistoryRepository: historyRepository,
			},
		}
	case "grok":
		model = &grok_model.GrokModel{OpenAICompatibleModel: openai_base.OpenAICompatibleModel{ResponseHandler: responseHandler, HistoryRepository: historyRepository}}
	case "4o":
		model = &openai_4o_model.OpenAi4oModel{ResponseHandler: responseHandler, HistoryRepository: historyRepository}
	case "gpt":
		model = &open_ai_gpt_model.OpenAIGPTModel{OpenAICompatibleModel: openai_base.OpenAICompatibleModel{ResponseHandler: responseHandler, HistoryRepository: historyRepository}, ModelVersion: "gpt"}
	case "codex":
		model = &open_ai_gpt_model.OpenAIGPTModel{OpenAICompatibleModel: openai_base.OpenAICompatibleModel{ResponseHandler: responseHandler, HistoryRepository: historyRepository}, ModelVersion: "codex"}
	case "ollama":
		model = ollama_model.NewOllamaModel(responseHandler, "")
	case "qwen3":
		model = ollama_model.NewOllamaModel(responseHandler, "")
	case "opus":
		model = &claude_model.ClaudeModel{UseStreaming: streamMode, HistoryRepository: historyRepository, ResponseHandler: responseHandler, UseThinking: thinkingMode, StreamThought: streamThinkingMode, OutputThought: outputThinkingMode, ModelVersion: "opus"}
	case "sonnet":
		model = &claude_model.ClaudeModel{UseStreaming: streamMode, HistoryRepository: historyRepository, ResponseHandler: responseHandler, UseThinking: thinkingMode, StreamThought: streamThinkingMode, OutputThought: outputThinkingMode, ModelVersion: "sonnet"}
	case "haiku":
		model = &claude_model.ClaudeModel{UseStreaming: streamMode, HistoryRepository: historyRepository, ResponseHandler: responseHandler, UseThinking: thinkingMode, StreamThought: streamThinkingMode, OutputThought: outputThinkingMode, ModelVersion: "haiku"}
	case "claude":
		fallthrough
	default:
		model = &claude_model.ClaudeModel{UseStreaming: streamMode, HistoryRepository: historyRepository, ResponseHandler: responseHandler, UseThinking: thinkingMode, StreamThought: streamThinkingMode, OutputThought: outputThinkingMode}
		modelToUse = "claude"
	}

	return model, modelToUse
}
