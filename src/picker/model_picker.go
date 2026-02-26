package models

import (
	"fmt"
	commontypes "owl/common_types"
	"owl/data"
	"owl/logger"
	claude_model "owl/models/claude"
	grok_model "owl/models/grok"
	ollama_model "owl/models/ollama"
	openai_4o_model "owl/models/open-ai-4o"
	openai_base "owl/models/open-ai-base"
	open_ai_gpt_model "owl/models/open-ai-gpt"

	"github.com/fatih/color"
)

func getModelForQuery(
	requestedModel string,
	context *data.Context,
	responseHandler commontypes.ResponseHandler,
	historyRepository data.HistoryRepository,
	streamMode bool,
) (commontypes.Model, string) {
	logger.Screen(fmt.Sprintf("Getting model for request. Requested model: %s", requestedModel), color.RGB(150, 150, 150))
	modelToUse := requestedModel

	if modelToUse == "" && context != nil && context.PreferredModel != "" {
		modelToUse = context.PreferredModel
	}

	if modelToUse == "" {
		modelToUse = "claude"
	}

	var model commontypes.Model

	switch modelToUse {
	case "grok":
		model = &grok_model.GrokModel{
			OpenAICompatibleModel: openai_base.OpenAICompatibleModel{
				ResponseHandler:   responseHandler,
				HistoryRepository: historyRepository,
			},
		}
	case "4o":
		model = &openai_4o_model.OpenAi4oModel{
			ResponseHandler:   responseHandler,
			HistoryRepository: historyRepository,
		}
	case "gpt":
		model = &open_ai_gpt_model.OpenAIGPTModel{
			OpenAICompatibleModel: openai_base.OpenAICompatibleModel{
				ResponseHandler:   responseHandler,
				HistoryRepository: historyRepository,
			},
			ModelVersion: "gpt",
		}
	case "codex":
		model = &open_ai_gpt_model.OpenAIGPTModel{
			OpenAICompatibleModel: openai_base.OpenAICompatibleModel{
				ResponseHandler:   responseHandler,
				HistoryRepository: historyRepository,
			},
			ModelVersion: "codex",
		}
	case "ollama":
		model = ollama_model.NewOllamaModel(responseHandler, "")
	case "qwen3":
		model = ollama_model.NewOllamaModel(responseHandler, "")
	case "opus":
		model = &claude_model.ClaudeModel{
			UseStreaming:      streamMode,
			HistoryRepository: historyRepository,
			ResponseHandler:   responseHandler,
			ModelVersion:      "opus",
		}
	case "sonnet":
		model = &claude_model.ClaudeModel{
			UseStreaming:      streamMode,
			HistoryRepository: historyRepository,
			ResponseHandler:   responseHandler,
			ModelVersion:      "sonnet",
		}
	case "haiku":
		model = &claude_model.ClaudeModel{
			UseStreaming:      streamMode,
			HistoryRepository: historyRepository,
			ResponseHandler:   responseHandler,
			ModelVersion:      "haiku",
		}
	case "claude":
		fallthrough
	default:
		model = &claude_model.ClaudeModel{
			UseStreaming:      streamMode,
			HistoryRepository: historyRepository,
			ResponseHandler:   responseHandler,
		}
		modelToUse = "claude"
	}

	logger.Screen(fmt.Sprintf("selected model %s for request. Requested model: %s", modelToUse, requestedModel), color.RGB(150, 150, 150))

	return model, modelToUse
}

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
