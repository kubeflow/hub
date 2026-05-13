package mocks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	"github.com/kubeflow/model-registry/pkg/openapi"
)

func GetRegisteredModelMocks() []openapi.RegisteredModel {
	model1 := openapi.RegisteredModel{
		CustomProperties:         newCustomProperties(),
		Name:                     "Model One",
		Description:              stringToPointer("This model does things and stuff"),
		ExternalId:               stringToPointer("934589798"),
		Id:                       stringToPointer("1"),
		CreateTimeSinceEpoch:     stringToPointer("1725282249921"),
		LastUpdateTimeSinceEpoch: stringToPointer("1725282249921"),
		Owner:                    stringToPointer("Sherlock Holmes"),
		State:                    stateToPointer(openapi.REGISTEREDMODELSTATE_LIVE),
	}

	model2 := openapi.RegisteredModel{
		CustomProperties:         newCustomProperties(),
		Name:                     "Model Two",
		Description:              stringToPointer("This model does things and stuff"),
		ExternalId:               stringToPointer("345235987"),
		Id:                       stringToPointer("2"),
		CreateTimeSinceEpoch:     stringToPointer("1725282249921"),
		LastUpdateTimeSinceEpoch: stringToPointer("1725282249921"),
		Owner:                    stringToPointer("John Watson"),
		State:                    stateToPointer(openapi.REGISTEREDMODELSTATE_LIVE),
	}

	model3 := openapi.RegisteredModel{
		CustomProperties:         newCustomProperties(),
		Name:                     "Model Three",
		Description:              stringToPointer("This model does things and stuff"),
		ExternalId:               stringToPointer("345235989"),
		Id:                       stringToPointer("3"),
		CreateTimeSinceEpoch:     stringToPointer("1725282249933"),
		LastUpdateTimeSinceEpoch: stringToPointer("1725282249933"),
		Owner:                    stringToPointer("M. Oriarty"),
		State:                    stateToPointer(openapi.REGISTEREDMODELSTATE_ARCHIVED),
	}

	return []openapi.RegisteredModel{model1, model2, model3}
}

func GetRegisteredModelListMock() openapi.RegisteredModelList {
	models := GetRegisteredModelMocks()

	return openapi.RegisteredModelList{
		NextPageToken: "abcdefgh",
		PageSize:      2,
		Size:          int32(len(models)),
		Items:         models,
	}
}

func GetModelVersionMocks() []openapi.ModelVersion {
	modelVersion1 := openapi.ModelVersion{
		CustomProperties:         newCustomProperties(),
		Name:                     "Version One",
		Description:              stringToPointer("This version improves stuff and things"),
		ExternalId:               stringToPointer("934589798"),
		Id:                       stringToPointer("1"),
		CreateTimeSinceEpoch:     stringToPointer("1725282249921"),
		LastUpdateTimeSinceEpoch: stringToPointer("1725282249921"),
		RegisteredModelId:        "1",
		Author:                   stringToPointer("Sherlock Holmes"),
		State:                    stateToPointer(openapi.MODELVERSIONSTATE_LIVE),
	}

	modelVersion2 := openapi.ModelVersion{
		CustomProperties:         newCustomProperties(),
		Name:                     "Version Two",
		Description:              stringToPointer("This version improves stuff and things better"),
		ExternalId:               stringToPointer("934589798"),
		Id:                       stringToPointer("2"),
		CreateTimeSinceEpoch:     stringToPointer("1725282259922"),
		LastUpdateTimeSinceEpoch: stringToPointer("1725282259922"),
		RegisteredModelId:        "1",
		Author:                   stringToPointer("Sherlock Holmes"),
		State:                    stateToPointer(openapi.MODELVERSIONSTATE_LIVE),
	}

	modelVersion3 := openapi.ModelVersion{
		CustomProperties:         newCustomProperties(),
		Name:                     "Version Three",
		Description:              stringToPointer("This version improves stuff and things"),
		ExternalId:               stringToPointer("934589799"),
		Id:                       stringToPointer("3"),
		CreateTimeSinceEpoch:     stringToPointer("1725282249921"),
		LastUpdateTimeSinceEpoch: stringToPointer("1725282249921"),
		RegisteredModelId:        "2",
		Author:                   stringToPointer("Sherlock Holmes"),
		State:                    stateToPointer(openapi.MODELVERSIONSTATE_LIVE),
	}

	modelVersion4 := openapi.ModelVersion{
		CustomProperties:         newCustomProperties(),
		Name:                     "Version Four",
		Description:              stringToPointer("This version didn't improve stuff and things"),
		ExternalId:               stringToPointer("934589791"),
		Id:                       stringToPointer("4"),
		CreateTimeSinceEpoch:     stringToPointer("1725282249921"),
		LastUpdateTimeSinceEpoch: stringToPointer("1725282249921"),
		RegisteredModelId:        "3",
		Author:                   stringToPointer("Sherlock Holmes"),
		State:                    stateToPointer(openapi.MODELVERSIONSTATE_ARCHIVED),
	}

	return []openapi.ModelVersion{modelVersion1, modelVersion2, modelVersion3, modelVersion4}
}

func GetModelVersionListMock() openapi.ModelVersionList {
	versions := GetModelVersionMocks()

	return openapi.ModelVersionList{
		NextPageToken: "abcdefgh",
		PageSize:      2,
		Items:         versions,
		Size:          2,
	}
}

func GetModelArtifactMocks() []openapi.ModelArtifact {
	artifact1 := openapi.ModelArtifact{
		ArtifactType:             stringToPointer("TYPE_ONE"),
		CustomProperties:         newCustomProperties(),
		Description:              stringToPointer("This artifact can do more than you would expect"),
		ExternalId:               stringToPointer("1000001"),
		Uri:                      stringToPointer("oci://quay.io/my-org/my-model:v1.0.0"),
		State:                    stateToPointer(openapi.ARTIFACTSTATE_LIVE),
		Name:                     stringToPointer("Artifact One"),
		Id:                       stringToPointer("1"),
		CreateTimeSinceEpoch:     stringToPointer("1725282249921"),
		LastUpdateTimeSinceEpoch: stringToPointer("1725282249921"),
		ModelFormatName:          stringToPointer("ONNX"),
		StorageKey:               stringToPointer("key1"),
		StoragePath:              stringToPointer("/artifacts/1"),
		ModelFormatVersion:       stringToPointer("1.0.0"),
		ServiceAccountName:       stringToPointer("service-1"),
		ModelSourceKind:          stringToPointer("transfer_job"),
		ModelSourceGroup:         stringToPointer("bella-namespace"),
		ModelSourceName:          stringToPointer("transfer-job-001"),
	}

	artifact2 := openapi.ModelArtifact{
		ArtifactType:             stringToPointer("TYPE_TWO"),
		CustomProperties:         newCustomProperties(),
		Description:              stringToPointer("This artifact can do more than you would expect, but less than you would hope"),
		ExternalId:               stringToPointer("1000002"),
		Uri:                      stringToPointer("http://localhost/artifacts/2"),
		State:                    stateToPointer(openapi.ARTIFACTSTATE_PENDING),
		Name:                     stringToPointer("Artifact Two"),
		Id:                       stringToPointer("2"),
		CreateTimeSinceEpoch:     stringToPointer("1725282249921"),
		LastUpdateTimeSinceEpoch: stringToPointer("1725282249921"),
		ModelFormatName:          stringToPointer("TensorFlow"),
		StorageKey:               stringToPointer("key2"),
		StoragePath:              stringToPointer("/artifacts/2"),
		ModelFormatVersion:       stringToPointer("1.0.0"),
		ServiceAccountName:       stringToPointer("service-2"),
	}

	return []openapi.ModelArtifact{artifact1, artifact2}
}

func GetModelArtifactListMock() openapi.ModelArtifactList {
	return openapi.ModelArtifactList{
		NextPageToken: "abcdefgh",
		PageSize:      2,
		Items:         GetModelArtifactMocks(),
		Size:          2,
	}
}

func newCustomProperties() *map[string]openapi.MetadataValue {
	result := map[string]openapi.MetadataValue{
		"tensorflow": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "",
				MetadataType: "MetadataStringValue",
			},
		},
		"pytorch": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "",
				MetadataType: "MetadataStringValue",
			},
		},
		"mll": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "",
				MetadataType: "MetadataStringValue",
			},
		},
		"rnn": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "",
				MetadataType: "MetadataStringValue",
			},
		},
		"AWS_KEY": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "asdf89asdf098asdfa",
				MetadataType: "MetadataStringValue",
			},
		},
		"AWS_PASSWORD": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "*AadfeDs34adf",
				MetadataType: "MetadataStringValue",
			},
		},
	}

	return &result
}

func catalogCustomProperties() *map[string]openapi.MetadataValue {
	return catalogCustomPropertiesWithVariant("", "FP16")
}

func catalogCustomPropertiesWithVariant(variantGroupId string, tensorType string) *map[string]openapi.MetadataValue {
	result := map[string]openapi.MetadataValue{
		"tensorflow": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "",
				MetadataType: "MetadataStringValue",
			},
		},
		"pytorch": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "",
				MetadataType: "MetadataStringValue",
			},
		},
		"mll": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "",
				MetadataType: "MetadataStringValue",
			},
		},
		"rnn": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "",
				MetadataType: "MetadataStringValue",
			},
		},
		"validated": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "",
				MetadataType: "MetadataStringValue",
			},
		},
		"validated_on": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "[\"RHOAI 2.20\",\"RHAIIS 3.0\",\"RHELAI 1.5\"]",
				MetadataType: "MetadataStringValue",
			},
		},
		"AWS_KEY": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "asdf89asdf098asdfa",
				MetadataType: "MetadataStringValue",
			},
		},
		"AWS_PASSWORD": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "*AadfeDs34adf",
				MetadataType: "MetadataStringValue",
			},
		},
		"tensor_type": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  tensorType,
				MetadataType: "MetadataStringValue",
			},
		},
		"size": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "8B params",
				MetadataType: "MetadataStringValue",
			},
		},
		"model_type": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "generative",
				MetadataType: "MetadataStringValue",
			},
		},
	}

	// Add variant_group_id if provided
	if variantGroupId != "" {
		result["variant_group_id"] = openapi.MetadataValue{
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  variantGroupId,
				MetadataType: "MetadataStringValue",
			},
		}
	}

	return &result
}

func NewMockSessionContext(parent context.Context) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	traceId := uuid.NewString()
	ctx := context.WithValue(parent, constants.TraceIdKey, traceId)

	traceLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = context.WithValue(ctx, constants.TraceLoggerKey, traceLogger)
	return ctx
}

func NewMockSessionContextNoParent() context.Context {
	return NewMockSessionContext(context.TODO())
}

func GenerateMockArtifactList() openapi.ArtifactList {
	var artifacts []openapi.Artifact
	for i := 0; i < 2; i++ {
		artifact := GenerateMockArtifact()
		artifacts = append(artifacts, artifact)
	}

	return openapi.ArtifactList{
		NextPageToken: gofakeit.UUID(),
		PageSize:      int32(gofakeit.Number(1, 20)),
		Size:          int32(len(artifacts)),
		Items:         artifacts,
	}
}

func GenerateMockArtifact() openapi.Artifact {
	modelArtifact := GenerateMockModelArtifact()

	mockData := openapi.Artifact{
		ModelArtifact: &modelArtifact,
	}
	return mockData
}

const graniteVariantGroupId = "b6c850a4-aa4c-4a0f-91b1-0a69f4352843"

func GetCatalogModelMocks() []models.CatalogModel {
	sampleModel1 := models.CatalogModel{
		Name:             "repo1/granite-8b-code-instruct",
		Description:      stringToPointer("Granite-8B-Code-Instruct is a 8B parameter model fine tuned from\nGranite-8B-Code-Base on a combination of permissively licensed instruction\ndata to enhance instruction following capabilities including logical\nreasoning and problem-solving skills."),
		Provider:         stringToPointer("provider1"),
		Tasks:            []string{"text-generation", "image-to-text", "tool-calling"},
		ValidatedTasks:   []string{"tool-calling"},
		License:          stringToPointer("Apache 2.0"),
		LicenseLink:      stringToPointer("https://www.apache.org/licenses/LICENSE-2.0.txt"),
		Maturity:         stringToPointer("Technology preview"),
		Language:         []string{"ar", "cs", "de", "en", "es", "fr", "it", "ja", "ko", "nl", "pt", "zh"},
		CustomProperties: catalogCustomPropertiesWithVariant(graniteVariantGroupId, "FP16"),
		ServingConfig: &models.ServingConfig{
			ToolCalling: &models.ToolCallingConfig{
				Args: stringToPointer("--enable-auto-tool-choice \\\n--tool-call-parser granite \\\n--chat-template\nopt/app-root/template/tool_chat_template_granite.jinja"),
			},
		},
		Logo: stringToPointer("data:image/svg+xml;base64,PHN2ZyBpZD0iTGF5ZXJfMSIgZGF0YS1uYW1lPSJMYXllciAxIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAxOTIgMTQ1Ij48ZGVmcz48c3R5bGU+LmNscy0xe2ZpbGw6I2UwMDt9PC9zdHlsZT48L2RlZnM+PHRpdGxlPlJlZEhhdC1Mb2dvLUhhdC1Db2xvcjwvdGl0bGU+PHBhdGggZD0iTTE1Ny43Nyw2Mi42MWExNCwxNCwwLDAsMSwuMzEsMy40MmMwLDE0Ljg4LTE4LjEsMTcuNDYtMzAuNjEsMTcuNDZDNzguODMsODMuNDksNDIuNTMsNTMuMjYsNDIuNTMsNDRhNi40Myw2LjQzLDAsMCwxLC4yMi0xLjk0bC0zLjY2LDkuMDZhMTguNDUsMTguNDUsMCwwLDAtMS41MSw3LjMzYzAsMTguMTEsNDEsNDUuNDgsODcuNzQsNDUuNDgsMjAuNjksMCwzNi40My03Ljc2LDM2LjQzLTIxLjc3LDAtMS4wOCwwLTEuOTQtMS43My0xMC4xM1oiLz48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik0xMjcuNDcsODMuNDljMTIuNTEsMCwzMC42MS0yLjU4LDMwLjYxLTE3LjQ2YTE0LDE0LDAsMCwwLS4zMS0zLjQybC03LjQ1LTMyLjM2Yy0xLjcyLTcuMTItMy4yMy0xMC4zNS0xNS43My0xNi42QzEyNC44OSw4LjY5LDEwMy43Ni41LDk3LjUxLjUsOTEuNjkuNSw5MCw4LDgzLjA2LDhjLTYuNjgsMC0xMS42NC01LjYtMTcuODktNS42LTYsMC05LjkxLDQuMDktMTIuOTMsMTIuNSwwLDAtOC40MSwyMy43Mi05LjQ5LDI3LjE2QTYuNDMsNi40MywwLDAsMCw0Mi41Myw0NGMwLDkuMjIsMzYuMywzOS40NSw4NC45NCwzOS40NU0xNjAsNzIuMDdjMS43Myw4LjE5LDEuNzMsOS4wNSwxLjczLDEwLjEzLDAsMTQtMTUuNzQsMjEuNzctMzYuNDMsMjEuNzdDNzguNTQsMTA0LDM3LjU4LDc2LjYsMzcuNTgsNTguNDlhMTguNDUsMTguNDUsMCwwLDEsMS41MS03LjMzQzIyLjI3LDUyLC41LDU1LC41LDc0LjIyYzAsMzEuNDgsNzQuNTksNzAuMjgsMTMzLjY1LDcwLjI4LDQ1LjI4LDAsNTYuNy0yMC40OCw1Ni43LTM2LjY1LDAtMTIuNzItMTEtMjcuMTYtMzAuODMtMzUuNzgiLz48L3N2Zz4="),
		Readme: stringToPointer(`---
pipeline_tag: text-generation
inference: false
license: apache-2.0
library_name: transformers
tags:
- language
- granite-3.1
base_model:
- provider1-granite/granite-3.1-8b-base
---

# Granite-3.1-8B-Instruct

**Model Summary:**
Granite-3.1-8B-Instruct is a 8B parameter long-context instruct model finetuned from Granite-3.1-8B-Base using a combination of open source instruction datasets with permissive license and internally collected synthetic datasets tailored for solving long context problems. This model is developed using a diverse set of techniques with a structured chat format, including supervised finetuning, model alignment using reinforcement learning, and model merging.

- **Developers:** Granite Team, provider1
- **GitHub Repository:** [provider1-granite/granite-3.1-language-models](https://github.com/provider1-granite/granite-3.1-language-models)
- **Website**: [Granite Docs](https://www.provider1.com/granite/docs/)
- **Paper:** [Granite 3.1 Language Models (coming soon)](https://huggingface.co/collections/provider1-granite/granite-31-language-models-6751dbbf2f3389bec5c6f02d) 
- **Release Date**: December 18th, 2024
- **License:** [Apache 2.0](https://www.apache.org/licenses/LICENSE-2.0)

**Supported Languages:** 
English, German, Spanish, French, Japanese, Portuguese, Arabic, Czech, Italian, Korean, Dutch, and Chinese. Users may finetune Granite 3.1 models for languages beyond these 12 languages.

**Intended Use:** 
The model is designed to respond to general instructions and can be used to build AI assistants for multiple domains, including business applications.

*Capabilities*
* Summarization
* Text classification
* Text extraction
* Question-answering
* Retrieval Augmented Generation (RAG)
* Code related tasks
* Function-calling tasks
* Multilingual dialog use cases
* Long-context tasks including long document/meeting summarization, long document QA, etc.

**Generation:** 
This is a simple example of how to use Granite-3.1-8B-Instruct model.

Install the following libraries:

` + "```" + `shell
pip install torch torchvision torchaudio
pip install accelerate
pip install transformers
` + "```" + `
Then, copy the snippet from the section that is relevant for your use case.

 ` + "```" + `python
import torch
from transformers import AutoModelForCausalLM, AutoTokenizer

device = "auto"
model_path = "provider1-granite/granite-3.1-8b-instruct"
tokenizer = AutoTokenizer.from_pretrained(model_path)
# drop device_map if running on CPU
model = AutoModelForCausalLM.from_pretrained(model_path, device_map=device)
model.eval()
# change input text as desired
chat = [
    { "role": "user", "content": "Please list one provider1 Research laboratory located in the United States. You should only output its name and location." },
]
chat = tokenizer.apply_chat_template(chat, tokenize=False, add_generation_prompt=True)
# tokenize the text
input_tokens = tokenizer(chat, return_tensors="pt").to(device)
# generate output tokens
output = model.generate(**input_tokens, 
                        max_new_tokens=100)
# decode output tokens into text
output = tokenizer.batch_decode(output)
# print output
print(output)
` + "```" + `

**Evaluation Results:**
<table>
  <caption><b>HuggingFace Open LLM Leaderboard V1</b></caption>
<thead>
  <tr>
    <th style="text-align:left; background-color: #001d6c; color: white;">Models</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">ARC-Challenge</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">Hellaswag</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">MMLU</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">TruthfulQA</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">Winogrande</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">GSM8K</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">Avg</th>
  </tr></thead>
  <tbody>
  <tr>
    <td style="text-align:left; background-color: #DAE8FF; color: black;">Granite-3.1-8B-Instruct</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">62.62</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">84.48</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">65.34</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">66.23</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">75.37</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">73.84</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">71.31</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">Granite-3.1-2B-Instruct</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">54.61</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">75.14</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">55.31</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">59.42</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">67.48</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">52.76</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">60.79</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">Granite-3.1-3B-A800M-Instruct</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">50.42</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">73.01</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">52.19</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">49.71</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">64.87</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">48.97</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">56.53</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">Granite-3.1-1B-A400M-Instruct</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">42.66</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">65.97</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">26.13</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">46.77</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">62.35</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">33.88</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">46.29</td>
  </tr>
</tbody></table>

<table>
  <caption><b>HuggingFace Open LLM Leaderboard V2</b></caption>
<thead>
  <tr>
    <th style="text-align:left; background-color: #001d6c; color: white;">Models</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">IFEval</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">BBH</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">MATH Lvl 5</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">GPQA</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">MUSR</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">MMLU-Pro</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">Avg</th>
  </tr></thead>
  <tbody>
  <tr>
    <td style="text-align:left; background-color: #DAE8FF; color: black;">Granite-3.1-8B-Instruct</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">72.08</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">34.09</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">21.68</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">8.28</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">19.01</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">28.19</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">30.55</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">Granite-3.1-2B-Instruct</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">62.86</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">21.82</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">11.33</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">5.26</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">4.87</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">20.21</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">21.06</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">Granite-3.1-3B-A800M-Instruct</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">55.16</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">16.69</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">10.35</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">5.15</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">2.51</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">12.75</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">17.1</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">Granite-3.1-1B-A400M-Instruct</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">46.86</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">6.18</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">4.08</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">0</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">0.78</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">2.41</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">10.05</td>
  </tr>
</tbody></table>

**Model Architecture:**
Granite-3.1-8B-Instruct is based on a decoder-only dense transformer architecture. Core components of this architecture are: GQA and RoPE, MLP with SwiGLU, RMSNorm, and shared input/output embeddings.

<table>
<thead>
  <tr>
    <th style="text-align:left; background-color: #001d6c; color: white;">Model</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">2B Dense</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">8B Dense</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">1B MoE</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">3B MoE</th>
  </tr></thead>
<tbody>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Embedding size</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">2048</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">4096</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">1024</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">1536</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Number of layers</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">40</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">40</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">24</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">32</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Attention head size</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">64</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">128</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">64</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">64</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Number of attention heads</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">32</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">32</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">16</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">24</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Number of KV heads</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">8</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">8</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">8</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">8</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">MLP hidden size</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">8192</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">12800</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">512</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">512</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">MLP activation</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">SwiGLU</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">SwiGLU</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">SwiGLU</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">SwiGLU</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Number of experts</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">—</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">—</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">32</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">40</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">MoE TopK</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">—</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">—</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">8</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">8</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Initialization std</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">0.1</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">0.1</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">0.1</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">0.1</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Sequence length</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">128K</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">128K</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">128K</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">128K</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Position embedding</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">RoPE</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">RoPE</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">RoPE</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">RoPE</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;"># Parameters</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">2.5B</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">8.1B</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">1.3B</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">3.3B</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;"># Active parameters</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">2.5B</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">8.1B</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">400M</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">800M</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;"># Training tokens</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">12T</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">12T</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">10T</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">10T</td>
  </tr>
</tbody></table>

**Training Data:** 
Overall, our SFT data is largely comprised of three key sources: (1) publicly available datasets with permissive license, (2) internal synthetic data targeting specific capabilities including long-context tasks, and (3) very small amounts of human-curated data. A detailed attribution of datasets can be found in the [Granite 3.0 Technical Report](https://github.com/provider1-granite/granite-3.0-language-models/blob/main/paper.pdf), [Granite 3.1 Technical Report (coming soon)](https://huggingface.co/collections/provider1-granite/granite-31-language-models-6751dbbf2f3389bec5c6f02d), and [Accompanying Author List](https://github.com/provider1-granite/granite-3.0-language-models/blob/main/author-ack.pdf).

**Infrastructure:**
We train Granite 3.1 Language Models using provider1's super computing cluster, Blue Vela, which is outfitted with NVIDIA H100 GPUs. This cluster provides a scalable and efficient infrastructure for training our models over thousands of GPUs.

**Ethical Considerations and Limitations:** 
Granite 3.1 Instruct Models are primarily finetuned using instruction-response pairs mostly in English, but also multilingual data covering eleven languages. Although this model can handle multilingual dialog use cases, its performance might not be similar to English tasks. In such case, introducing a small number of examples (few-shot) can help the model in generating more accurate outputs. While this model has been aligned by keeping safety in consideration, the model may in some cases produce inaccurate, biased, or unsafe responses to user prompts. So we urge the community to use this model with proper safety testing and tuning tailored for their specific tasks.

**Resources**
- ⭐️ Learn about the latest updates with Granite: https://www.provider1.com/granite
- 📄 Get started with tutorials, best practices, and prompt engineering advice: https://www.provider1.com/granite/docs/
- 💡 Learn about the latest Granite learning resources: https://provider1.biz/granite-learning-resources

<!-- ## Citation
` + "```" + `
@misc{granite-models,
  author = {author 1, author2, ...},
  title = {},
  journal = {},
  volume = {},
  year = {2024},
  url = {https://arxiv.org/abs/0000.00000},
}
  ` + "```" + ` -->`),
		SourceId:                 stringToPointer("sample-source"),
		LibraryName:              stringToPointer("transformers"),
		CreateTimeSinceEpoch:     stringToPointer("1693526400000"),
		LastUpdateTimeSinceEpoch: stringToPointer("1704067200000"),
	}

	sampleModel2 := models.CatalogModel{
		Name:             "repo1/granite-8b-code-instruct-quantized.w4a16",
		Description:      stringToPointer("Granite 8B Code Instruct - INT4 quantized variant for efficient inference"),
		Provider:         stringToPointer("Provider one"),
		Tasks:            []string{"text-generation", "image-text-to-text"},
		License:          stringToPointer("Apache 2.0"),
		Maturity:         stringToPointer("Generally Available"),
		Language:         []string{"en"},
		SourceId:         stringToPointer("sample-source"),
		CustomProperties: catalogCustomPropertiesWithVariant(graniteVariantGroupId, "INT4"),
		Logo:             stringToPointer("data:image/svg+xml;base64,PHN2ZyBpZD0iTGF5ZXJfMSIgZGF0YS1uYW1lPSJMYXllciAxIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAxOTIgMTQ1Ij48ZGVmcz48c3R5bGU+LmNscy0xe2ZpbGw6I2UwMDt9PC9zdHlsZT48L2RlZnM+PHRpdGxlPlJlZEhhdC1Mb2dvLUhhdC1Db2xvcjwvdGl0bGU+PHBhdGggZD0iTTE1Ny43Nyw2Mi42MWExNCwxNCwwLDAsMSwuMzEsMy40MmMwLDE0Ljg4LTE4LjEsMTcuNDYtMzAuNjEsMTcuNDZDNzguODMsODMuNDksNDIuNTMsNTMuMjYsNDIuNTMsNDRhNi40Myw2LjQzLDAsMCwxLC4yMi0xLjk0bC0zLjY2LDkuMDZhMTguNDUsMTguNDUsMCwwLDAtMS41MSw3LjMzYzAsMTguMTEsNDEsNDUuNDgsODcuNzQsNDUuNDgsMjAuNjksMCwzNi40My03Ljc2LDM2LjQzLTIxLjc3LDAtMS4wOCwwLTEuOTQtMS43My0xMC4xM1oiLz48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik0xMjcuNDcsODMuNDljMTIuNTEsMCwzMC42MS0yLjU4LDMwLjYxLTE3LjQ2YTE0LDE0LDAsMCwwLS4zMS0zLjQybC03LjQ1LTMyLjM2Yy0xLjcyLTcuMTItMy4yMy0xMC4zNS0xNS43My0xNi42QzEyNC44OSw4LjY5LDEwMy43Ni41LDk3LjUxLjUsOTEuNjkuNSw5MCw4LDgzLjA2LDhjLTYuNjgsMC0xMS42NC01LjYtMTcuODktNS42LTYsMC05LjkxLDQuMDktMTIuOTMsMTIuNSwwLDAtOC40MSwyMy43Mi05LjQ5LDI3LjE2QTYuNDMsNi40MywwLDAsMCw0Mi41Myw0NGMwLDkuMjIsMzYuMywzOS40NSw4NC45NCwzOS40NU0xNjAsNzIuMDdjMS43Myw4LjE5LDEuNzMsOS4wNSwxLjczLDEwLjEzLDAsMTQtMTUuNzQsMjEuNzctMzYuNDMsMjEuNzdDNzguNTQsMTA0LDM3LjU4LDc2LjYsMzcuNTgsNTguNDlhMTguNDUsMTguNDUsMCwwLDEsMS41MS03LjMzQzIyLjI3LDUyLC41LDU1LC41LDc0LjIyYzAsMzEuNDgsNzQuNTksNzAuMjgsMTMzLjY1LDcwLjI4LDQ1LjI4LDAsNTYuNy0yMC40OCw1Ni43LTM2LjY1LDAtMTIuNzItMTEtMjcuMTYtMzAuODMtMzUuNzgiLz48L3N2Zz4="),
	}

	sampleModel3 := models.CatalogModel{
		Name:             "repo1/granite-8b-code-instruct-quantized.w8a8",
		Description:      stringToPointer("Granite 8B Code Instruct - INT8 quantized variant for balanced performance"),
		Provider:         stringToPointer("IBM"),
		Tasks:            []string{"audio-to-text", "text-to-text", "video-to-text"},
		License:          stringToPointer("MIT"),
		Maturity:         stringToPointer("Generally Available"),
		Language:         []string{"en"},
		SourceId:         stringToPointer("sample-source"),
		CustomProperties: catalogCustomPropertiesWithVariant(graniteVariantGroupId, "INT8"),
		Logo:             stringToPointer("data:image/svg+xml;base64,PHN2ZyBpZD0iTGF5ZXJfMSIgZGF0YS1uYW1lPSJMYXllciAxIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAxOTIgMTQ1Ij48ZGVmcz48c3R5bGU+LmNscy0xe2ZpbGw6I2UwMDt9PC9zdHlsZT48L2RlZnM+PHRpdGxlPlJlZEhhdC1Mb2dvLUhhdC1Db2xvcjwvdGl0bGU+PHBhdGggZD0iTTE1Ny43Nyw2Mi42MWExNCwxNCwwLDAsMSwuMzEsMy40MmMwLDE0Ljg4LTE4LjEsMTcuNDYtMzAuNjEsMTcuNDZDNzguODMsODMuNDksNDIuNTMsNTMuMjYsNDIuNTMsNDRhNi40Myw2LjQzLDAsMCwxLC4yMi0xLjk0bC0zLjY2LDkuMDZhMTguNDUsMTguNDUsMCwwLDAtMS41MSw3LjMzYzAsMTguMTEsNDEsNDUuNDgsODcuNzQsNDUuNDgsMjAuNjksMCwzNi40My03Ljc2LDM2LjQzLTIxLjc3LDAtMS4wOCwwLTEuOTQtMS43My0xMC4xM1oiLz48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik0xMjcuNDcsODMuNDljMTIuNTEsMCwzMC42MS0yLjU4LDMwLjYxLTE3LjQ2YTE0LDE0LDAsMCwwLS4zMS0zLjQybC03LjQ1LTMyLjM2Yy0xLjcyLTcuMTItMy4yMy0xMC4zNS0xNS43My0xNi42QzEyNC44OSw4LjY5LDEwMy43Ni41LDk3LjUxLjUsOTEuNjkuNSw5MCw4LDgzLjA2LDhjLTYuNjgsMC0xMS42NC01LjYtMTcuODktNS42LTYsMC05LjkxLDQuMDktMTIuOTMsMTIuNSwwLDAtOC40MSwyMy43Mi05LjQ5LDI3LjE2QTYuNDMsNi40MywwLDAsMCw0Mi41Myw0NGMwLDkuMjIsMzYuMywzOS40NSw4NC45NCwzOS40NU0xNjAsNzIuMDdjMS43Myw4LjE5LDEuNzMsOS4wNSwxLjczLDEwLjEzLDAsMTQtMTUuNzQsMjEuNzctMzYuNDMsMjEuNzdDNzguNTQsMTA0LDM3LjU4LDc2LjYsMzcuNTgsNTguNDlhMTguNDUsMTguNDUsMCwwLDEsMS41MS03LjMzQzIyLjI3LDUyLC41LDU1LC41LDc0LjIyYzAsMzEuNDgsNzQuNTksNzAuMjgsMTMzLjY1LDcwLjI4LDQ1LjI4LDAsNTYuNy0yMC40OCw1Ni43LTM2LjY1LDAtMTIuNzItMTEtMjcuMTYtMzAuODMtMzUuNzgiLz48L3N2Zz4="),
	}

	sampleModel4 := models.CatalogModel{
		Name:             "repo1/granite-8b-code-instruct-bf16",
		Description:      stringToPointer("Granite 8B Code Instruct - BF16 variant for high precision"),
		Provider:         stringToPointer("IBM"),
		Tasks:            []string{"text-generation", "code-generation"},
		License:          stringToPointer("Apache 2.0"),
		Maturity:         stringToPointer("Generally Available"),
		Language:         []string{"en"},
		SourceId:         stringToPointer("sample-source"),
		CustomProperties: catalogCustomPropertiesWithVariant(graniteVariantGroupId, "BF16"),
		Logo:             stringToPointer("data:image/svg+xml;base64,PHN2ZyBpZD0iTGF5ZXJfMSIgZGF0YS1uYW1lPSJMYXllciAxIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAxOTIgMTQ1Ij48ZGVmcz48c3R5bGU+LmNscy0xe2ZpbGw6I2UwMDt9PC9zdHlsZT48L2RlZnM+PHRpdGxlPlJlZEhhdC1Mb2dvLUhhdC1Db2xvcjwvdGl0bGU+PHBhdGggZD0iTTE1Ny43Nyw2Mi42MWExNCwxNCwwLDAsMSwuMzEsMy40MmMwLDE0Ljg4LTE4LjEsMTcuNDYtMzAuNjEsMTcuNDZDNzguODMsODMuNDksNDIuNTMsNTMuMjYsNDIuNTMsNDRhNi40Myw2LjQzLDAsMCwxLC4yMi0xLjk0bC0zLjY2LDkuMDZhMTguNDUsMTguNDUsMCwwLDAtMS41MSw3LjMzYzAsMTguMTEsNDEsNDUuNDgsODcuNzQsNDUuNDgsMjAuNjksMCwzNi40My03Ljc2LDM2LjQzLTIxLjc3LDAtMS4wOCwwLTEuOTQtMS43My0xMC4xM1oiLz48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik0xMjcuNDcsODMuNDljMTIuNTEsMCwzMC42MS0yLjU4LDMwLjYxLTE3LjQ2YTE0LDE0LDAsMCwwLS4zMS0zLjQybC03LjQ1LTMyLjM2Yy0xLjcyLTcuMTItMy4yMy0xMC4zNS0xNS43My0xNi42QzEyNC44OSw4LjY5LDEwMy43Ni41LDk3LjUxLjUsOTEuNjkuNSw5MCw4LDgzLjA2LDhjLTYuNjgsMC0xMS42NC01LjYtMTcuODktNS42LTYsMC05LjkxLDQuMDktMTIuOTMsMTIuNSwwLDAtOC40MSwyMy43Mi05LjQ5LDI3LjE2QTYuNDMsNi40MywwLDAsMCw0Mi41Myw0NGMwLDkuMjIsMzYuMywzOS40NSw4NC45NCwzOS40NU0xNjAsNzIuMDdjMS43Myw4LjE5LDEuNzMsOS4wNSwxLjczLDEwLjEzLDAsMTQtMTUuNzQsMjEuNzctMzYuNDMsMjEuNzdDNzguNTQsMTA0LDM3LjU4LDc2LjYsMzcuNTgsNTguNDlhMTguNDUsMTguNDUsMCwwLDEsMS41MS03LjMzQzIyLjI3LDUyLC41LDU1LC41LDc0LjIyYzAsMzEuNDgsNzQuNTksNzAuMjgsMTMzLjY1LDcwLjI4LDQ1LjI4LDAsNTYuNy0yMC40OCw1Ni43LTM2LjY1LDAtMTIuNzItMTEtMjcuMTYtMzAuODMtMzUuNzgiLz48L3N2Zz4="),
	}

	huggingFaceModel1 := models.CatalogModel{
		Name:        "provider2/bert-base-uncased",
		Description: stringToPointer("BERT base model (uncased) - Pretrained model on English language"),
		Provider:    stringToPointer("Google"),
		Tasks:       []string{"audio-to-text", "text-to-text"},
		License:     stringToPointer("Apache 2.0"),
		Maturity:    stringToPointer("Generally Available"),
		Language:    []string{"en"},
		SourceId:    stringToPointer("huggingface"),
		LibraryName: stringToPointer("transformers"),
		Readme: stringToPointer(`# BERT Base Uncased

BERT is a transformers model pretrained on a large corpus of English data.

## Installation

` + "```bash" + `
pip install transformers torch
` + "```" + `

## Quick Start

` + "```python" + `
from transformers import BertTokenizer, BertModel

tokenizer = BertTokenizer.from_pretrained('bert-base-uncased')
model = BertModel.from_pretrained('bert-base-uncased')

text = "Replace this with your text"
encoded = tokenizer(text, return_tensors='pt')
output = model(**encoded)
` + "```" + `

## Using with Pipeline

` + "```bash" + `
python -c "from transformers import pipeline; nlp = pipeline('fill-mask', model='bert-base-uncased'); print(nlp('The capital of France is [MASK].'))"
` + "```" + `
`),
	}

	huggingFaceModel2 := models.CatalogModel{
		Name:        "provider3/gpt2",
		Description: stringToPointer("GPT-2 is a transformers model pretrained on a very large corpus of English data"),
		Provider:    stringToPointer("provider3"),
		Tasks:       []string{"video-to-text"},
		License:     stringToPointer("MIT"),
		Maturity:    stringToPointer("Generally Available"),
		Language:    []string{"en"},
		SourceId:    stringToPointer("huggingface"),
		LibraryName: stringToPointer("transformers"),
	}

	huggingFaceModel3 := models.CatalogModel{
		Name:        "huggingface/distilbert-base-uncased",
		Description: stringToPointer("DistilBERT base model (uncased) - A smaller, faster version of BERT"),
		Provider:    stringToPointer("Hugging Face"),
		Tasks:       []string{"fill-mask", "text-classification"},
		License:     stringToPointer("Apache 2.0"),
		Maturity:    stringToPointer("Generally Available"),
		Language:    []string{"en"},
		SourceId:    stringToPointer("huggingface"),
		LibraryName: stringToPointer("transformers"),
	}

	otherModel1 := models.CatalogModel{
		Name:        "adminModel2/admin-model-1",
		Description: stringToPointer("sample description"),
		Provider:    stringToPointer("Admin model 1"),
		Tasks:       []string{"code-generation", "instruction-following"},
		License:     stringToPointer("Apache 2.0"),
		Maturity:    stringToPointer("Generally Available"),
		Language:    []string{"en"},
		SourceId:    stringToPointer("adminModel2"),
	}

	otherModel2 := models.CatalogModel{
		Name:        "adminModel1/admin-model-2",
		Description: stringToPointer("sample description"),
		Provider:    stringToPointer("Admin model 2"),
		Tasks:       []string{"text-generation", "conversational"},
		License:     stringToPointer("Apache 2.0"),
		Maturity:    stringToPointer("Generally Available"),
		Language:    []string{"en"},
		SourceId:    stringToPointer("adminModel1"),
	}

	noPerformanceModel := models.CatalogModel{
		Name:        "no-perf-source/test-model",
		Description: stringToPointer("Model without performance data"),
		Provider:    stringToPointer("Test Provider"),
		Tasks:       []string{"text-generation"},
		License:     stringToPointer("Apache 2.0"),
		Language:    []string{"en"},
		SourceId:    stringToPointer("no-perf-source"),
	}

	// added this to test the load more models button
	var additionalRepo1Models []models.CatalogModel
	for i := 1; i <= 20; i++ {
		model := models.CatalogModel{
			Name:                     fmt.Sprintf("repo1/granite-model-%d", i),
			Description:              stringToPointer("Granite-8B-Code-Instruct is a 8B parameter model fine tuned from\nGranite-8B-Code-Base on a combination of permissively licensed instruction\ndata to enhance instruction following capabilities including logical\nreasoning and problem-solving skills."),
			Provider:                 stringToPointer("provider1"),
			Tasks:                    []string{"text-generation"},
			License:                  stringToPointer("Apache 2.0"),
			LicenseLink:              stringToPointer("https://www.apache.org/licenses/LICENSE-2.0.txt"),
			Maturity:                 stringToPointer("Technology preview"),
			Language:                 []string{"ar", "cs", "de", "en", "es", "fr", "it", "ja", "ko", "nl", "pt", "zh"},
			Logo:                     stringToPointer("data:image/svg+xml;base64,PHN2ZyBpZD0iTGF5ZXJfMSIgZGF0YS1uYW1lPSJMYXllciAxIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAxOTIgMTQ1Ij48ZGVmcz48c3R5bGU+LmNscy0xe2ZpbGw6I2UwMDt9PC9zdHlsZT48L2RlZnM+PHRpdGxlPlJlZEhhdC1Mb2dvLUhhdC1Db2xvcjwvdGl0bGU+PHBhdGggZD0iTTE1Ny43Nyw2Mi42MWExNCwxNCwwLDAsMSwuMzEsMy40MmMwLDE0Ljg4LTE4LjEsMTcuNDYtMzAuNjEsMTcuNDZDNzguODMsODMuNDksNDIuNTMsNTMuMjYsNDIuNTMsNDRhNi40Myw2LjQzLDAsMCwxLC4yMi0xLjk0bC0zLjY2LDkuMDZhMTguNDUsMTguNDUsMCwwLDAtMS41MSw3LjMzYzAsMTguMTEsNDEsNDUuNDgsODcuNzQsNDUuNDgsMjAuNjksMCwzNi40My03Ljc2LDM2LjQzLTIxLjc3LDAtMS4wOCwwLTEuOTQtMS43My0xMC4xM1oiLz48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik0xMjcuNDcsODMuNDljMTIuNTEsMCwzMC42MS0yLjU4LDMwLjYxLTE3LjQ2YTE0LDE0LDAsMCwwLS4zMS0zLjQybC03LjQ1LTMyLjM2Yy0xLjcyLTcuMTItMy4yMy0xMC4zNS0xNS43My0xNi42QzEyNC44OSw4LjY5LDEwMy43Ni41LDk3LjUxLjUsOTEuNjkuNSw5MCw4LDgzLjA2LDhjLTYuNjgsMC0xMS42NC01LjYtMTcuODktNS42LTYsMC05LjkxLDQuMDktMTIuOTMsMTIuNSwwLDAtOC40MSwyMy43Mi05LjQ5LDI3LjE2QTYuNDMsNi40MywwLDAsMCw0Mi41Myw0NGMwLDkuMjIsMzYuMywzOS40NSw4NC45NCwzOS40NU0xNjAsNzIuMDdjMS43Myw4LjE5LDEuNzMsOS4wNSwxLjczLDEwLjEzLDAsMTQtMTUuNzQsMjEuNzctMzYuNDMsMjEuNzdDNzguNTQsMTA0LDM3LjU4LDc2LjYsMzcuNTgsNTguNDlhMTguNDUsMTguNDUsMCwwLDEsMS41MS03LjMzQzIyLjI3LDUyLC41LDU1LC41LDc0LjIyYzAsMzEuNDgsNzQuNTksNzAuMjgsMTMzLjY1LDcwLjI4LDQ1LjI4LDAsNTYuNy0yMC40OCw1Ni43LTM2LjY1LDAtMTIuNzItMTEtMjcuMTYtMzAuODMtMzUuNzgiLz48L3N2Zz4="),
			SourceId:                 stringToPointer("sample-source"),
			LibraryName:              stringToPointer("transformers"),
			CreateTimeSinceEpoch:     stringToPointer("1693526400000"),
			LastUpdateTimeSinceEpoch: stringToPointer("1704067200000"),
			CustomProperties:         catalogCustomProperties(),
		}
		additionalRepo1Models = append(additionalRepo1Models, model)
	}

	allModels := []models.CatalogModel{
		sampleModel1, sampleModel2, sampleModel3, sampleModel4,
		huggingFaceModel1, huggingFaceModel2, huggingFaceModel3, noPerformanceModel,
		otherModel1, otherModel2,
	}
	allModels = append(allModels, additionalRepo1Models...)

	return allModels
}

func GetCatalogModelListMock() models.CatalogModelList {
	allModels := GetCatalogModelMocks()

	return models.CatalogModelList{
		Items:         allModels,
		Size:          int32(len(allModels)),
		PageSize:      int32(10),
		NextPageToken: "10",
	}
}

func GetCatalogSourceMocks() []models.CatalogSource {
	enabled := true
	disabledBool := false

	// Status examples (matching OpenAPI spec)
	availableStatus := "available"
	errorStatus := "error"
	disabledStatus := "disabled"

	invalidCredentialError := "The provided API key is invalid or has expired. Please update your credentials."
	invalidOrgError := "The specified organization 'invalid-org' does not exist or you don't have access to it. Please verify the organization name and ensure you have the necessary permissions to access models from this organization."

	return []models.CatalogSource{
		{
			Id:      "sample-source",
			Name:    "Sample mocked source",
			Enabled: &enabled,
			Labels:  []string{"Sample category 1", "Sample category 2", "Sample category"},
			Status:  &availableStatus,
		},
		{
			Id:     "huggingface",
			Name:   "Hugging Face",
			Labels: []string{"Sample category 2", "Sample category"},
			// Status is nil - represents "Starting" state (no status yet)
			Status: nil,
		},
		{
			Id:      "adminModel1",
			Name:    "Admin model 1",
			Enabled: &enabled,
			Labels:  []string{},
			Status:  &errorStatus,
			Error:   &invalidCredentialError,
		},
		{
			Id:      "adminModel2",
			Name:    "Admin model 2",
			Enabled: &enabled,
			Labels:  []string{"Sample category 1"},
			Status:  &errorStatus,
			Error:   &invalidOrgError,
		},
		{
			Id:     "dora",
			Name:   "Dora source",
			Labels: []string{},
			Status: &availableStatus,
		},
		{
			Id:      "adminModel3",
			Name:    "Admin model 3",
			Enabled: &disabledBool,
			Labels:  []string{},
			Status:  &disabledStatus,
		},
		{
			Id:      "no-perf-source",
			Name:    "No Performance Data Source",
			Enabled: &enabled,
			Labels:  []string{"No Performance"},
			Status:  &availableStatus,
		},
		{
			Id:      "custom_yaml_models",
			Name:    "Custom yaml",
			Enabled: &enabled,
			Labels:  []string{},
			Status:  &errorStatus,
			Error:   &invalidCredentialError,
		},
		{
			Id:      "hugging_face_source",
			Name:    "Hugging face source",
			Enabled: &enabled,
			Labels:  []string{},
			Status:  &availableStatus,
		},
	}
}

func GetCatalogSourceListMock() models.CatalogSourceList {
	allSources := GetCatalogSourceMocks()

	return models.CatalogSourceList{
		Items:         allSources,
		Size:          int32(len(allSources)),
		PageSize:      int32(10),
		NextPageToken: "",
	}
}

func GetCatalogLabelListMock() models.CatalogLabelList {
	redHatAI := "Red Hat AI"
	redHatAIValidated := "Red Hat AI Validated"
	sampleCategory1 := "Sample category 1"
	sampleCategory2 := "Sample category 2"
	community := "Community"
	otherModels := "Other models"

	redHatAIDisplay := "Red Hat AI models"
	redHatAIValidatedDisplay := "Red Hat AI Validated models"
	sampleCategory1Display := "Sample Category 1"
	sampleCategory2Display := "Sample Category 2"
	communityDisplay := "Community models"

	redHatAIDesc := "Red Hat AI models are curated and optimized for performance on Red Hat platforms."
	redHatAIValidatedDesc := "Validated models are benchmarked for performance and quality using leading open source evaluation datasets."
	sampleCategory1Desc := "Sample category 1 description"
	sampleCategory2Desc := "Sample category 2 description"
	communityDesc := "Community contributed models from various sources."
	otherModelsDesc := "Models without a specific category label."

	labels := []models.CatalogLabel{
		{
			Name:        &redHatAI,
			DisplayName: &redHatAIDisplay,
			Description: &redHatAIDesc,
		},
		{
			Name:        &redHatAIValidated,
			DisplayName: &redHatAIValidatedDisplay,
			Description: &redHatAIValidatedDesc,
		},
		{
			Name:        &sampleCategory1,
			DisplayName: &sampleCategory1Display,
			Description: &sampleCategory1Desc,
		},
		{
			Name:        &sampleCategory2,
			DisplayName: &sampleCategory2Display,
			Description: &sampleCategory2Desc,
		},
		{
			Name:        &community,
			DisplayName: &communityDisplay,
			Description: &communityDesc,
		},
		{
			Name:        nil,
			DisplayName: &otherModels,
			Description: &otherModelsDesc,
		},
	}

	return models.CatalogLabelList{
		Items:         labels,
		Size:          int32(len(labels)),
		PageSize:      int32(10),
		NextPageToken: "",
	}
}

func GetCatalogModelArtifactMock() []models.CatalogArtifact {
	architecturesJSON, _ := json.Marshal([]string{"amd64", "arm64", "s390x", "ppc64le"})
	customProps := newCustomProperties()
	(*customProps)["architecture"] = openapi.MetadataValue{
		MetadataStringValue: &openapi.MetadataStringValue{
			StringValue:  string(architecturesJSON),
			MetadataType: "MetadataStringValue",
		},
	}

	return []models.CatalogArtifact{
		{
			ArtifactType:             "model-artifact",
			Uri:                      stringToPointer("oci://registry.sample.io/repo1/modelcar-granite-7b-starter:1.4.0"),
			CreateTimeSinceEpoch:     stringToPointer("1693526400000"),
			LastUpdateTimeSinceEpoch: stringToPointer("1704067200000"),
			CustomProperties:         customProps,
		},
	}
}

func performanceMetricsCustomProperties(customProperties map[string]openapi.MetadataValue) *map[string]openapi.MetadataValue {
	result := map[string]openapi.MetadataValue{
		"config_id": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "0055d94f6547237dgf324238",
				MetadataType: "MetadataStringValue",
			},
		},
		"ttft_mean": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  35.48818160947744,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"ttft_p90": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  51.55777931213379,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"ttft_p95": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  61.26761436462402,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"ttft_p99": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  72.95823097229004,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"e2e_mean": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  1994.480013381083,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"e2e_p90": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  2644.604682922363,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"e2e_p95": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  2813.79246711731,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"e2e_p99": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  3117.565155029297,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"tps_mean": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  1785.325259154939,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"tps_p90": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  3318.2751083374023,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"tps_p95": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  4934.475563049316,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"tps_p99": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  11781.748535156249,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"itl_mean": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  7.6874115515873105,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"itl_p90": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  7.782459259033203,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"itl_p95": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  7.808256149291992,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"itl_p99": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  7.911920547485352,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"requests_per_second": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  7,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"max_input_tokens": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  1024,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"max_output_tokens": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  1,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"hardware_type": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "H100",
				MetadataType: "MetadataStringValue",
			},
		},
		"hardware_count": {
			MetadataIntValue: &openapi.MetadataIntValue{
				IntValue:     "2",
				MetadataType: "MetadataIntValue",
			},
		},
		"framework": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "vllm",
				MetadataType: "MetadataStringValue",
			},
		},
		"framework_version": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "v0.1.1",
				MetadataType: "MetadataStringValue",
			},
		},
		"docker_image": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "vllm/vllm-openai:v0.1.1",
				MetadataType: "MetadataStringValue",
			},
		},
		"entrypoint": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue: "\npython3\n",
			},
		},
		"inserted_at": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "2025-05-07T00:00:00.000Z",
				MetadataType: "MetadataStringValue",
			},
		},
		"created_at": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "2025-05-07T00:00:00.000Z",
				MetadataType: "MetadataStringValue",
			},
		},
		"updated_at": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue:  "2025-05-14T12:08:25.402Z",
				MetadataType: "MetadataStringValue",
			},
		},
		"mean_input_tokens": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  511.5445458496306,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"mean_output_tokens": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  255.8678835289005,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"model_hf_repo_name": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue: "provider1-granite/granite-3.1-8b-instruct",
			},
		},
		"replicas": {
			MetadataIntValue: &openapi.MetadataIntValue{
				IntValue:     "3",
				MetadataType: "MetadataIntValue",
			},
		},
		"total_requests_per_second": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  150.0,
				MetadataType: "MetadataDoubleValue",
			},
		},
	}
	for key, value := range customProperties {
		result[key] = value
	}
	return &result
}

func GetCatalogPerformanceMetricsArtifactMock(itemCount int32) []models.CatalogArtifact {
	artifacts := []models.CatalogArtifact{
		{
			ArtifactType:             *stringToPointer("metrics-artifact"),
			MetricsType:              stringToPointer("performance-metrics"),
			CreateTimeSinceEpoch:     stringToPointer("1693526400000"),
			LastUpdateTimeSinceEpoch: stringToPointer("1704067200000"),
			CustomProperties: performanceMetricsCustomProperties(map[string]openapi.MetadataValue{
				"config_id": {
					MetadataStringValue: &openapi.MetadataStringValue{
						StringValue:  "config-001-chatbot-h100",
						MetadataType: "MetadataStringValue",
					},
				},
				"use_case": {
					MetadataStringValue: &openapi.MetadataStringValue{
						StringValue:  "chatbot",
						MetadataType: "MetadataStringValue",
					},
				},
			}),
		},
		{
			ArtifactType:             *stringToPointer("metrics-artifact"),
			MetricsType:              stringToPointer("performance-metrics"),
			CreateTimeSinceEpoch:     stringToPointer("1693526400000"),
			LastUpdateTimeSinceEpoch: stringToPointer("1704067200000"),
			CustomProperties: performanceMetricsCustomProperties(map[string]openapi.MetadataValue{
				"config_id": {
					MetadataStringValue: &openapi.MetadataStringValue{
						StringValue:  "config-002-rag-rtx4090",
						MetadataType: "MetadataStringValue",
					},
				},
				"hardware_type": {
					MetadataStringValue: &openapi.MetadataStringValue{
						StringValue:  "RTX 4090",
						MetadataType: "MetadataStringValue",
					},
				},
				"hardware_count": {
					MetadataIntValue: &openapi.MetadataIntValue{
						IntValue:     "4",
						MetadataType: "MetadataIntValue",
					},
				},
				"requests_per_second": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  10,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"ttft_mean": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  67.15382947561234,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"ttft_p90": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  82.34921756823456,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"ttft_p95": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  95.67834521987654,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"ttft_p99": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  112.45678234561234,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"e2e_mean": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  2450.32847561234123,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"e2e_p90": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  3120.45678912345678,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"e2e_p95": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  3450.78234567891234,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"e2e_p99": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  3890.12567891234567,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"itl_mean": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  9.458723456123456,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"itl_p90": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  11.23456789123456,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"itl_p95": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  13.56789123456789,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"itl_p99": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  16.78912345678901,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"use_case": {
					MetadataStringValue: &openapi.MetadataStringValue{
						StringValue:  "rag",
						MetadataType: "MetadataStringValue",
					},
				},
			}),
		},
		{
			ArtifactType:             *stringToPointer("metrics-artifact"),
			MetricsType:              stringToPointer("performance-metrics"),
			CreateTimeSinceEpoch:     stringToPointer("1693526400000"),
			LastUpdateTimeSinceEpoch: stringToPointer("1704067200000"),
			CustomProperties: performanceMetricsCustomProperties(map[string]openapi.MetadataValue{
				"config_id": {
					MetadataStringValue: &openapi.MetadataStringValue{
						StringValue:  "config-003-codefixing-a100",
						MetadataType: "MetadataStringValue",
					},
				},
				"hardware_type": {
					MetadataStringValue: &openapi.MetadataStringValue{
						StringValue:  "A100",
						MetadataType: "MetadataStringValue",
					},
				},
				"hardware_count": {
					MetadataIntValue: &openapi.MetadataIntValue{
						IntValue:     "8",
						MetadataType: "MetadataIntValue",
					},
				},
				"requests_per_second": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  15,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"ttft_mean": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  42.12834756189234,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"ttft_p90": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  58.45912378456123,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"ttft_p95": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  68.92345678901234,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"ttft_p99": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  85.34567891234567,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"e2e_mean": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  1850.67891234567891,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"e2e_p90": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  2280.34567891234567,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"e2e_p95": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  2580.91234567891234,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"e2e_p99": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  2920.45678912345678,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"itl_mean": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  6.78912345678901,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"itl_p90": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  8.12345678912345,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"itl_p95": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  9.45678912345678,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"itl_p99": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  11.23456789123456,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"use_case": {
					MetadataStringValue: &openapi.MetadataStringValue{
						StringValue:  "code_fixing",
						MetadataType: "MetadataStringValue",
					},
				},
			}),
		},
		{
			ArtifactType:             *stringToPointer("metrics-artifact"),
			MetricsType:              stringToPointer("performance-metrics"),
			CreateTimeSinceEpoch:     stringToPointer("1693526400000"),
			LastUpdateTimeSinceEpoch: stringToPointer("1704067200000"),
			CustomProperties: performanceMetricsCustomProperties(map[string]openapi.MetadataValue{
				"config_id": {
					MetadataStringValue: &openapi.MetadataStringValue{
						StringValue:  "config-004-longrag-a100",
						MetadataType: "MetadataStringValue",
					},
				},
				"hardware_type": {
					MetadataStringValue: &openapi.MetadataStringValue{
						StringValue:  "A100-80",
						MetadataType: "MetadataStringValue",
					},
				},
				"hardware_count": {
					MetadataIntValue: &openapi.MetadataIntValue{
						IntValue:     "2",
						MetadataType: "MetadataIntValue",
					},
				},
				"requests_per_second": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  25,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"ttft_mean": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  28.50789123456789,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"ttft_p90": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  38.72345678912345,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"ttft_p95": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  45.89123456789012,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"ttft_p99": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  55.32456789123456,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"e2e_mean": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  1450.23456789123456,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"e2e_p90": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  1780.45678912345678,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"e2e_p95": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  1980.67891234567891,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"e2e_p99": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  2250.89123456789012,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"itl_mean": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  5.23456789123456,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"itl_p90": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  6.45678912345678,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"itl_p95": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  7.23456789123456,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"itl_p99": {
					MetadataDoubleValue: &openapi.MetadataDoubleValue{
						DoubleValue:  8.56789123456789,
						MetadataType: "MetadataDoubleValue",
					},
				},
				"use_case": {
					MetadataStringValue: &openapi.MetadataStringValue{
						StringValue:  "long_rag",
						MetadataType: "MetadataStringValue",
					},
				},
			}),
		},
	}
	artifacts = artifacts[:itemCount]
	return artifacts
}

func accuracyMetricsCustomProperties() *map[string]openapi.MetadataValue {
	result := map[string]openapi.MetadataValue{
		"overall_average": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  0.584329,
				MetadataType: "MetadataDoubleValue",
			},
		},
		"arc_v1": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue:  0.674673,
				MetadataType: "MetadataDoubleValue",
			},
		},
	}
	return &result
}

func GetCatalogAccuracyMetricsArtifactMock() []models.CatalogArtifact {
	return []models.CatalogArtifact{
		{
			ArtifactType:             *stringToPointer("metrics-artifact"),
			MetricsType:              stringToPointer("accuracy-metrics"),
			CreateTimeSinceEpoch:     stringToPointer("1693526400000"),
			LastUpdateTimeSinceEpoch: stringToPointer("1704067200000"),
			CustomProperties:         accuracyMetricsCustomProperties(),
		},
	}
}
func GetModelArtifactListMockWithItems(items []models.CatalogArtifact, pageSize int32) models.CatalogModelArtifactList {
	return models.CatalogModelArtifactList{
		Items:         items,
		Size:          int32(len(items)),
		PageSize:      pageSize,
		NextPageToken: "",
	}
}

func GetCatalogModelArtifactListMock() models.CatalogModelArtifactList {
	allArtifactMock := GetCatalogModelArtifactMock()
	return GetModelArtifactListMockWithItems(allArtifactMock, 10)
}

func GetCatalogPerformanceMetricsArtifactListMock(itemCount int32) models.CatalogModelArtifactList {
	allArtifactMock := GetCatalogPerformanceMetricsArtifactMock(itemCount)
	return GetModelArtifactListMockWithItems(allArtifactMock, 10)
}

func GetCatalogAccuracyMetricsArtifactListMock() models.CatalogModelArtifactList {
	allArtifactMock := GetCatalogAccuracyMetricsArtifactMock()
	return GetModelArtifactListMockWithItems(allArtifactMock, 10)
}

const (
	FilterOptionTypeString = "string"
	FilterOptionTypeNumber = "number"
)

func float32Ptr(i float32) *float32 {
	return &i
}

func GetFilterOptionMocks() map[string]models.FilterOption {
	filterOptions := make(map[string]models.FilterOption)

	filterOptions["provider"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"Alibaba Cloud", "DeepSeek", "Google", "IBM", "Meta", "Microsoft",
			"Mistral", "Mistral AI", "Moonshot AI", "NVIDIA", "Nvidia", "OpenAI", "Provider one",
		},
	}

	filterOptions["license"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"Apache 2.0", "Gemma License", "Llama 3.1 Community License",
			"Llama 3.3 Community License", "Llama 4 Community License", "MIT",
			"NVIDIA Open Model License", "Modified MIT",
		},
	}

	filterOptions["tasks"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"audio-to-text", "automatic-speech-recognition", "automatic-speech-translation",
			"code-generation", "image-text-to-text", "image-to-text", "text-embedding",
			"text-generation", "text-to-text", "tool-calling", "video-to-text",
		},
	}

	filterOptions["validatedTasks"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"tool-calling",
		},
	}

	filterOptions["language"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"ar", "bg", "ca", "cs", "da", "de", "el", "en", "es", "fa", "fi", "fr",
			"he", "hi", "hr", "hu", "id", "is", "it", "ja", "ko", "ms", "nl", "nld",
			"no", "pl", "pt", "ro", "ru", "sk", "sl", "sr", "sv", "th", "tl", "tr",
			"uk", "ur", "vi", "zh", "zsm",
		},
	}

	filterOptions["status"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"available", "disabled", "error",
		},
	}

	filterOptions["model_type.string_value"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"generative",
		},
	}

	filterOptions["size.string_value"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"108B params", "11B params", "120B params", "14B params", "19B params",
			"1B params", "1T params", "21.5B params", "23B params", "24B params",
			"2B params", "401B params", "46.7B params", "480B params", "4B params",
			"671B params", "7.62B params", "7.85B params", "70.6B params", "70B params",
			"7B params", "8 B", "8.19B params", "8.89B params", "8B params",
		},
	}

	filterOptions["tensor_type.string_value"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"FP16", "FP8", "INT4", "INT8", "MXFP4",
		},
	}

	filterOptions["validated_on.array_value"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"RHAIIS 3.0", "RHAIIS 3.2.1", "RHAIIS 3.2.2", "RHELAI 1.5",
			"RHOAI 2.20", "RHOAI 2.24", "RHOAI 2.25",
		},
	}

	filterOptions["variant_group_id.string_value"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"10bd75b4-ec27-4ee1-b0c8-5a0e41785b31", "2a4a5f11-fd59-4067-9be7-9d7a07a581c1",
			"36117278-8e53-4b44-9391-7ba28403caef", "6a1a6cce-efbc-4cea-b738-ea34ccf241a6",
		},
	}

	// Artifact properties (with artifacts. prefix)
	filterOptions["artifacts.use_case.string_value"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"chatbot", "code_fixing", "long_rag", "rag",
		},
	}

	filterOptions["artifacts.hardware_type.string_value"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"A100-40", "A100-80", "B200", "H100", "H200", "L4",
		},
	}

	filterOptions["artifacts.hardware_configuration.string_value"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"A100-40 x 1", "A100-40 x 2", "A100-40 x 4", "A100-40 x 8",
			"A100-80 x 1", "A100-80 x 2", "A100-80 x 4", "A100-80 x 8",
			"B200 x 1", "B200 x 2", "B200 x 4", "B200 x 8",
			"H100 x 1", "H100 x 2", "H100 x 4", "H100 x 8",
			"H200 x 1", "H200 x 2", "H200 x 4", "H200 x 8",
			"L4 x 1", "L4 x 2", "L4 x 4", "L4 x 8",
		},
	}

	filterOptions["artifacts.type.string_value"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"modelcar",
		},
	}

	filterOptions["artifacts.framework_type.string_value"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"vllm",
		},
	}

	filterOptions["artifacts.framework_version.string_value"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"0.8.4.20250429", "3.2.2", "rhoai-2.24-cuda-9c2c235775ca099889ee03dee8570b56df9d5d7e",
			"v0.10.1.1", "v0.8.4",
		},
	}

	filterOptions["artifacts.deployment_type.string_value"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"local",
		},
	}

	filterOptions["artifacts.source.string_value"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"sample.test.io",
		},
	}

	filterOptions["artifacts.tag.string_value"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"1.4", "1.4.0",
		},
	}

	filterOptions["artifacts.dataset.string_value"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"<nil>",
		},
	}

	// TTFT metrics
	filterOptions["artifacts.ttft_mean.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(15.928064),
			Max: float32Ptr(761.7213),
		},
	}

	filterOptions["artifacts.ttft_p90.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(16.912937),
			Max: float32Ptr(892.6554),
		},
	}

	filterOptions["artifacts.ttft_p95.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(17.714024),
			Max: float32Ptr(1149.7827),
		},
	}

	filterOptions["artifacts.ttft_p99.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(20.424604),
			Max: float32Ptr(4015.3562),
		},
	}

	// E2E metrics
	filterOptions["artifacts.e2e_mean.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(808.69165),
			Max: float32Ptr(66303.875),
		},
	}

	filterOptions["artifacts.e2e_p90.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(916.4949),
			Max: float32Ptr(76006.24),
		},
	}

	filterOptions["artifacts.e2e_p95.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(938.8428),
			Max: float32Ptr(79946.29),
		},
	}

	filterOptions["artifacts.e2e_p99.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(969.7657),
			Max: float32Ptr(87154.36),
		},
	}

	// ITL metrics
	filterOptions["artifacts.itl_mean.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(3.1054945),
			Max: float32Ptr(187.8361),
		},
	}

	filterOptions["artifacts.itl_p90.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(3.128499),
			Max: float32Ptr(223.09335),
		},
	}

	filterOptions["artifacts.itl_p95.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(3.1370418),
			Max: float32Ptr(224.09294),
		},
	}

	filterOptions["artifacts.itl_p99.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(3.1642547),
			Max: float32Ptr(225.55873),
		},
	}

	// TPS metrics
	filterOptions["artifacts.tps_mean.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(243.00464),
			Max: float32Ptr(8623.572),
		},
	}

	filterOptions["artifacts.tps_p90.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(240.04488),
			Max: float32Ptr(16578.277),
		},
	}

	filterOptions["artifacts.tps_p95.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(258.79584),
			Max: float32Ptr(24966.096),
		},
	}

	filterOptions["artifacts.tps_p99.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(285.5016),
			Max: float32Ptr(59918.63),
		},
	}

	// RPS and hardware count
	filterOptions["artifacts.requests_per_second.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(1),
			Max: float32Ptr(34),
		},
	}

	filterOptions["artifacts.hardware_count.int_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(1),
			Max: float32Ptr(8),
		},
	}

	// Token metrics
	filterOptions["artifacts.mean_input_tokens.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(501.44446),
			Max: float32Ptr(10240.305),
		},
	}

	filterOptions["artifacts.mean_output_tokens.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(254.25284),
			Max: float32Ptr(1539.3221),
		},
	}

	// Benchmark scores
	filterOptions["artifacts.aime24.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(63.3333),
			Max: float32Ptr(87.33),
		},
	}

	filterOptions["artifacts.aime25.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(56.6667),
			Max: float32Ptr(83.3333),
		},
	}

	filterOptions["artifacts.arc.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(58.5324),
			Max: float32Ptr(77.1331),
		},
	}

	filterOptions["artifacts.arc_challenge.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(58.2765),
			Max: float32Ptr(61.6041),
		},
	}

	filterOptions["artifacts.bbh.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(1.5589),
			Max: float32Ptr(70.9991),
		},
	}

	filterOptions["artifacts.gpqa.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(1.2988554),
			Max: float32Ptr(34.0855),
		},
	}

	filterOptions["artifacts.gpqa_diamond.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(30.8081),
			Max: float32Ptr(80.61),
		},
	}

	filterOptions["artifacts.gsm8k.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(1.3647),
			Max: float32Ptr(94.84),
		},
	}

	filterOptions["artifacts.hellaswag.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(47.4706),
			Max: float32Ptr(79.0829),
		},
	}

	filterOptions["artifacts.humaneval_instruct.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(98.0826),
			Max: float32Ptr(98.0826),
		},
	}

	filterOptions["artifacts.ifeval.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(57.3253),
			Max: float32Ptr(91.1024),
		},
	}

	filterOptions["artifacts.lcb.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(56.5998),
			Max: float32Ptr(56.5998),
		},
	}

	filterOptions["artifacts.math_500.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(95.475),
			Max: float32Ptr(97.4),
		},
	}

	filterOptions["artifacts.math_hard.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(16.7478),
			Max: float32Ptr(68.1613),
		},
	}

	filterOptions["artifacts.math_lvl5.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(24.7392),
			Max: float32Ptr(51.8965),
		},
	}

	filterOptions["artifacts.mgsm.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(28.1455),
			Max: float32Ptr(28.1455),
		},
	}

	filterOptions["artifacts.mmlu.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(62.8187),
			Max: float32Ptr(86.3125),
		},
	}

	filterOptions["artifacts.mmlu_pro.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(14.0625),
			Max: float32Ptr(64.0293),
		},
	}

	filterOptions["artifacts.musr.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(4.9206),
			Max: float32Ptr(46.8073),
		},
	}

	filterOptions["artifacts.overall_average.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(48.4109),
			Max: float32Ptr(98.0826),
		},
	}

	filterOptions["artifacts.truthfulqa.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(47.537),
			Max: float32Ptr(61.5658),
		},
	}

	filterOptions["artifacts.winogrande.double_value"] = models.FilterOption{
		Type: FilterOptionTypeNumber,
		Range: &models.FilterRange{
			Min: float32Ptr(61.9574),
			Max: float32Ptr(83.7411),
		},
	}

	return filterOptions
}

func GetNamedQueriesMocks() map[string]map[string]models.FieldFilter {
	namedQueries := make(map[string]map[string]models.FieldFilter)

	// Default performance filters - applied when performance toggle is turned on
	// Uses full filter key format matching the filters map
	namedQueries["default-performance-filters"] = map[string]models.FieldFilter{
		"artifacts.requests_per_second.double_value": {
			Operator: "<=",
			Value:    float64(1),
		},
		"artifacts.ttft_p90.double_value": {
			Operator: "<=",
			Value:    float64(892.6553726196289),
		},
		"artifacts.use_case.string_value": {
			Operator: "=",
			Value:    "chatbot",
		},
	}

	// Legacy validation-default query for backward compatibility
	namedQueries["validation-default"] = map[string]models.FieldFilter{
		"artifacts.ttft_p90.double_value": {
			Operator: "<",
			Value:    float64(70),
		},
		"artifacts.use_case.string_value": {
			Operator: "=",
			Value:    "chatbot",
		},
	}

	// High performance GPU configurations
	namedQueries["high_performance_gpu"] = map[string]models.FieldFilter{
		"artifacts.hardware_type.string_value": {
			Operator: "in",
			Value:    []interface{}{"H100-80", "A100-80"},
		},
		"artifacts.requests_per_second.double_value": {
			Operator: ">=",
			Value:    float64(50),
		},
	}

	// Low latency optimized
	namedQueries["low_latency"] = map[string]models.FieldFilter{
		"artifacts.ttft_p90.double_value": {
			Operator: "<",
			Value:    float64(100),
		},
		"artifacts.e2e_p90.double_value": {
			Operator: "<",
			Value:    float64(500),
		},
	}

	// Chatbot optimized
	namedQueries["chatbot_optimized"] = map[string]models.FieldFilter{
		"artifacts.use_case.string_value": {
			Operator: "=",
			Value:    "chatbot",
		},
	}

	// RAG optimized
	namedQueries["rag_optimized"] = map[string]models.FieldFilter{
		"artifacts.use_case.string_value": {
			Operator: "in",
			Value:    []interface{}{"rag", "long_rag"},
		},
	}

	return namedQueries
}

func GetFilterOptionsListMock() models.FilterOptionsList {
	filterOptions := GetFilterOptionMocks()
	namedQueries := GetNamedQueriesMocks()

	return models.FilterOptionsList{
		Filters:      &filterOptions,
		NamedQueries: &namedQueries,
	}
}

func BoolPtr(b bool) *bool {
	return &b
}

func GetModelsWithInclusionStatusListMocks() []models.CatalogSourcePreviewModel {
	// Generate enough models to test pagination (page size is 20)
	// We want 45 included and 25 excluded = 70 total models
	var allModels []models.CatalogSourcePreviewModel

	// Add 45 included models
	for i := 1; i <= 45; i++ {
		allModels = append(allModels, models.CatalogSourcePreviewModel{
			Name:     fmt.Sprintf("sample-source/included-model-%d", i),
			Included: true,
		})
	}

	// Add 25 excluded models
	for i := 1; i <= 25; i++ {
		allModels = append(allModels, models.CatalogSourcePreviewModel{
			Name:     fmt.Sprintf("sample-source/excluded-model-%d", i),
			Included: false,
		})
	}

	return allModels
}

func GetCatalogSourcePreviewSummaryMock() models.CatalogSourcePreviewSummary {
	return models.CatalogSourcePreviewSummary{
		TotalModels:    70,
		IncludedModels: 45,
		ExcludedModels: 25,
	}
}

func CreateCatalogSourcePreviewMock() models.CatalogSourcePreviewResult {
	return CreateCatalogSourcePreviewMockWithFilter("all", 20, "")
}

func CreateCatalogSourcePreviewMockWithFilter(filterStatus string, pageSize int, nextPageToken string) models.CatalogSourcePreviewResult {
	allModels := GetModelsWithInclusionStatusListMocks()
	catalogSourcePreviewSummary := GetCatalogSourcePreviewSummaryMock()

	// Filter based on filterStatus
	var filteredModels []models.CatalogSourcePreviewModel
	switch filterStatus {
	case "included":
		for _, m := range allModels {
			if m.Included {
				filteredModels = append(filteredModels, m)
			}
		}
	case "excluded":
		for _, m := range allModels {
			if !m.Included {
				filteredModels = append(filteredModels, m)
			}
		}
	default: // "all" or empty
		filteredModels = allModels
	}

	// Handle pagination
	startIndex := 0
	if nextPageToken != "" {
		// Parse token as start index (simple mock implementation)
		_, _ = fmt.Sscanf(nextPageToken, "%d", &startIndex)
	}

	if pageSize <= 0 {
		pageSize = 10
	}

	endIndex := startIndex + pageSize
	if endIndex > len(filteredModels) {
		endIndex = len(filteredModels)
	}

	pagedModels := filteredModels[startIndex:endIndex]

	// Generate next page token if there are more items
	var newNextPageToken string
	if endIndex < len(filteredModels) {
		newNextPageToken = fmt.Sprintf("%d", endIndex)
	}

	return models.CatalogSourcePreviewResult{
		Items:         pagedModels,
		Summary:       catalogSourcePreviewSummary,
		NextPageToken: newNextPageToken,
		PageSize:      int32(pageSize),
		Size:          int32(len(pagedModels)),
	}
}

func GetMcpServerMocks() []models.McpServer {
	trueVal := true
	falseVal := false

	prometheusMcp := models.McpServer{
		ID:          "1",
		Name:        "Prometheus MCP Server",
		SourceID:    stringToPointer("community-mcp-source"),
		Description: stringToPointer("Query Prometheus metrics and alerts directly from your agent"),
		Provider:    stringToPointer("Prometheus Community"),
		Version:     stringToPointer("0.9.2"),
		License:     stringToPointer("Apache 2.0"),
		LicenseLink: stringToPointer("https://www.apache.org/licenses/LICENSE-2.0"),
		Logo:        stringToPointer("https://raw.githubusercontent.com/cncf/artwork/main/projects/prometheus/icon/color/prometheus-icon-color.svg"),
		Tags:        []string{"metrics", "monitoring", "alerting"},
		ToolCount:   15,
		Transports:  []models.McpTransportType{models.McpTransportTypeHTTP},
		DeploymentMode: func() *models.McpDeploymentMode {
			mode := models.McpDeploymentModeLocal
			return &mode
		}(),
		SecurityIndicators: &models.McpSecurityIndicator{
			VerifiedSource: &trueVal,
			SecureEndpoint: &trueVal,
			SAST:           &falseVal,
			ReadOnlyTools:  &trueVal,
		},
		Tools: []models.McpTool{
			{
				Name:        "query",
				Description: stringToPointer("Execute PromQL queries"),
				AccessType:  models.McpToolAccessTypeReadOnly,
				Parameters: []models.McpToolParameter{
					{
						Name:        "query",
						Type:        "string",
						Description: stringToPointer("PromQL query expression"),
						Required:    true,
					},
				},
			},
		},
		Artifacts: []models.McpArtifact{
			{
				URI: "oci://ghcr.io/prometheus-community/prometheus-mcp:0.9.2",
			},
		},
		Readme:        stringToPointer("# Prometheus MCP Server\n\nThe Prometheus MCP Server enables AI assistants to query Prometheus metrics and alerts using natural language.\n\n## Quickstart\n\n```bash\nnpx -y @prometheus-community/prometheus-mcp-server\n```\n\n## Features\n\n- Execute PromQL queries\n- Retrieve active alerts\n- Query metric metadata\n- Range queries with configurable time windows\n"),
		SourceCode:    stringToPointer("prometheus-community/prometheus-mcp"),
		RepositoryURL: stringToPointer("https://github.com/prometheus-community/prometheus-mcp"),
		LastUpdated:   stringToPointer("1706745600000"),
	}

	kubernetesMcp := models.McpServer{
		ID:          "2",
		Name:        "Kubernetes MCP Server",
		SourceID:    stringToPointer("community-mcp-source"),
		Description: stringToPointer("Manage Kubernetes resources and query cluster state"),
		Provider:    stringToPointer("CNCF"),
		Version:     stringToPointer("1.2.0"),
		License:     stringToPointer("Apache 2.0"),
		LicenseLink: stringToPointer("https://www.apache.org/licenses/LICENSE-2.0"),
		Logo:        stringToPointer("https://raw.githubusercontent.com/kubernetes/kubernetes/master/logo/logo.svg"),
		Tags:        []string{"kubernetes", "containers", "orchestration"},
		ToolCount:   23,
		Transports:  []models.McpTransportType{models.McpTransportTypeHTTP, models.McpTransportTypeSSE},
		DeploymentMode: func() *models.McpDeploymentMode {
			mode := models.McpDeploymentModeLocal
			return &mode
		}(),
		SecurityIndicators: &models.McpSecurityIndicator{
			VerifiedSource: &trueVal,
			SecureEndpoint: &trueVal,
			SAST:           &trueVal,
			ReadOnlyTools:  &falseVal,
		},
		Artifacts: []models.McpArtifact{
			{
				URI: "oci://ghcr.io/cncf/kubernetes-mcp:1.2.0",
			},
		},
		Readme:        stringToPointer("# Kubernetes MCP Server\n\nThe Kubernetes MCP Server allows AI Assistants to interact with Kubernetes clusters, bringing real-time cluster management directly into your development workflow.\n\n**Note:** This product is not officially supported by CNCF.\n\nIf you need help, please contact us via GitHub Issues if you have feature requests, questions, or need help.\n\n## Quickstart\n\nYou can add this MCP server to your MCP Client like VSCode, Claude, Cursor, Amazon Q, Windsurf, ChatGPT, or GitHub Copilot via the command `npx -y @cncf/kubernetes-mcp-server` (type: stdio). For more details, please refer to the configuration section below.\n\n## Use Cases\n\n- **Real-time cluster management** - Query pod, deployment, and service status\n- **Resource operations** - Create, update, and delete Kubernetes resources\n- **Health monitoring** - Check cluster health and resource utilization\n- **Namespace management** - List and manage namespaces and their contents\n- **Debugging support** - Get pod logs and events for troubleshooting\n\n## Tools\n\n| Tool | Description |\n|------|-------------|\n| `get_pods` | List pods in a namespace |\n| `get_deployments` | List deployments |\n| `get_services` | List services |\n| `get_namespaces` | List all namespaces |\n| `get_pod_logs` | Retrieve pod logs |\n| `get_events` | Get cluster events |\n| `apply_manifest` | Apply a Kubernetes manifest |\n| `delete_resource` | Delete a Kubernetes resource |\n\n## Configuration\n\n```json\n{\n  \"mcpServers\": {\n    \"kubernetes\": {\n      \"command\": \"npx\",\n      \"args\": [\"-y\", \"@cncf/kubernetes-mcp-server\"],\n      \"env\": {\n        \"KUBECONFIG\": \"/path/to/kubeconfig\"\n      }\n    }\n  }\n}\n```\n\n## Requirements\n\n- Node.js 18+\n- Valid kubeconfig with cluster access\n- kubectl CLI (optional, for fallback operations)\n"),
		SourceCode:    stringToPointer("cncf/kubernetes-mcp-server"),
		RepositoryURL: stringToPointer("https://github.com/cncf/kubernetes-mcp-server"),
		LastUpdated:   stringToPointer("1709913600000"),
	}

	elasticMcp := models.McpServer{
		ID:          "3",
		Name:        "Elasticsearch MCP Server",
		SourceID:    stringToPointer("organization-mcp-source"),
		Description: stringToPointer("Search and analyze data in Elasticsearch clusters"),
		Provider:    stringToPointer("Elastic"),
		Version:     stringToPointer("2.0.5"),
		License:     stringToPointer("Elastic 2.0"),
		Tags:        []string{"search", "analytics", "observability"},
		ToolCount:   18,
		Transports:  []models.McpTransportType{models.McpTransportTypeHTTP},
		DeploymentMode: func() *models.McpDeploymentMode {
			mode := models.McpDeploymentModeRemote
			return &mode
		}(),
		SecurityIndicators: &models.McpSecurityIndicator{
			VerifiedSource: &trueVal,
			SecureEndpoint: &trueVal,
		},
		Endpoints: &models.McpEndpoints{
			HTTP: stringToPointer("https://api.mcpservers.org/elasticsearch-mcp/v1"),
			SSE:  stringToPointer("https://api.mcpservers.org/elasticsearch-mcp/sse"),
		},
	}

	dynatraceMcp := models.McpServer{
		ID:          "4",
		Name:        "Dynatrace MCP Server",
		SourceID:    stringToPointer("organization-mcp-source"),
		Description: stringToPointer("Access Dynatrace observability data and perform actions"),
		Provider:    stringToPointer("Dynatrace"),
		Version:     stringToPointer("1.0.1"),
		License:     stringToPointer("Apache 2.0"),
		LicenseLink: stringToPointer("https://github.com/dynatrace-oss/dynatrace-mcp-server/blob/main/LICENSE"),
		Tags:        []string{"observability", "monitoring", "apm"},
		ToolCount:   15,
		Transports:  []models.McpTransportType{models.McpTransportTypeStdio},
		DeploymentMode: func() *models.McpDeploymentMode {
			mode := models.McpDeploymentModeLocal
			return &mode
		}(),
		SecurityIndicators: &models.McpSecurityIndicator{
			VerifiedSource: &trueVal,
			SecureEndpoint: &trueVal,
			ReadOnlyTools:  &trueVal,
		},
		SourceCode:       stringToPointer("dynatrace-oss/dynatrace-mcp-server"),
		RepositoryURL:    stringToPointer("https://github.com/dynatrace-oss/dynatrace-mcp-server"),
		DocumentationURL: stringToPointer("https://github.com/dynatrace-oss/dynatrace-mcp-server/blob/main/README.md"),
		Readme:           stringToPointer("# Dynatrace MCP Server\n\nThe local Dynatrace MCP server allows AI Assistants to interact with the Dynatrace observability platform, bringing real-time observability data directly into your development workflow.\n\n**Note:** This product is not officially supported by Dynatrace.\n\nIf you need help, please contact us via GitHub Issues if you have feature requests, questions, or need help.\n\n## Quickstart\n\nYou can add this MCP server to your MCP Client like VSCode, Claude, Cursor, Amazon Q, Windsurf, ChatGPT, or GitHub Copilot via the command `npx -y @dynatrace-oss/dynatrace-mcp-server` (type: stdio). For more details, please refer to the configuration section below.\n\nFurthermore, you need to configure the URL to a Dynatrace environment:\n\n- `DT_ENVIRONMENT` (string, e.g., https://abc12345.apps.dynatrace.com) - URL to your Dynatrace Platform (do not use Dynatrace classic URLs like abc12345.live.dynatrace.com)\n\nOnce we are done, we recommend looking into example prompts, like \"Get all details of the entity 'my-service'\" or \"Show me error logs\". Please mind that these prompts lead to executing DQL statements which may incur costs in accordance to your licence.\n\n## Use Cases\n\n- **Real-time observability** - Fetch production-level data for early detection and proactive monitoring\n- **Contextual debugging** - Fix issues with full context from monitored exceptions, logs, and anomalies\n- **Security insights** - Get detailed vulnerability analysis and security problem tracking\n- **Natural language queries** - Use AI-powered DQL generation and explanation\n- **Multi-phase incident investigation** - Systematic 4-phase approach with automated impact assessment\n\n## Tools\n\n| Tool | Description |\n|------|-------------|\n| `execute_dql` | Execute Dynatrace Query Language (DQL) queries |\n| `get_problems` | Retrieve current problems and incidents |\n| `get_service_health` | Get health status of services |\n| `get_vulnerabilities` | Retrieve security vulnerability data |\n| `create_maintenance_window` | Create a maintenance window to suppress alerts |\n"),
		LastUpdated:      stringToPointer("1704067200000"),
	}

	grafanaMcp := models.McpServer{
		ID:          "5",
		Name:        "Grafana MCP Server",
		SourceID:    stringToPointer("community-mcp-source"),
		Description: stringToPointer("Query Grafana dashboards, data sources and annotations via natural language"),
		Provider:    stringToPointer("Grafana Labs"),
		Version:     stringToPointer("1.1.0"),
		License:     stringToPointer("AGPL-3.0"),
		LicenseLink: stringToPointer("https://www.gnu.org/licenses/agpl-3.0.html"),
		Logo:        stringToPointer("https://raw.githubusercontent.com/grafana/grafana/main/public/img/grafana_icon.svg"),
		Tags:        []string{"dashboards", "visualization", "monitoring"},
		ToolCount:   12,
		Transports:  []models.McpTransportType{models.McpTransportTypeHTTP},
		DeploymentMode: func() *models.McpDeploymentMode {
			mode := models.McpDeploymentModeRemote
			return &mode
		}(),
		SecurityIndicators: &models.McpSecurityIndicator{
			VerifiedSource: &trueVal,
			SecureEndpoint: &trueVal,
			ReadOnlyTools:  &trueVal,
		},
		Endpoints: &models.McpEndpoints{
			HTTP: stringToPointer("https://api.mcpservers.org/grafana-mcp/v1"),
		},
	}

	gitMcp := models.McpServer{
		ID:          "6",
		Name:        "Git MCP Server",
		SourceID:    stringToPointer("community-mcp-source"),
		Description: stringToPointer("Interact with Git repositories, branches, commits and diffs through your agent"),
		Provider:    stringToPointer("Git Community"),
		Version:     stringToPointer("0.5.3"),
		License:     stringToPointer("MIT"),
		LicenseLink: stringToPointer("https://opensource.org/licenses/MIT"),
		Tags:        []string{"git", "vcs", "repositories"},
		ToolCount:   20,
		Transports:  []models.McpTransportType{models.McpTransportTypeStdio},
		DeploymentMode: func() *models.McpDeploymentMode {
			mode := models.McpDeploymentModeLocal
			return &mode
		}(),
		SecurityIndicators: &models.McpSecurityIndicator{
			VerifiedSource: &trueVal,
			SAST:           &trueVal,
		},
	}

	postgresMcp := models.McpServer{
		ID:          "7",
		Name:        "PostgreSQL MCP Server",
		SourceID:    stringToPointer("organization-mcp-source"),
		Description: stringToPointer("Query and manage PostgreSQL databases using natural language"),
		Provider:    stringToPointer("PostgreSQL Global Development Group"),
		Version:     stringToPointer("1.3.0"),
		License:     stringToPointer("PostgreSQL License"),
		Tags:        []string{"database", "sql", "postgresql"},
		ToolCount:   10,
		Transports:  []models.McpTransportType{models.McpTransportTypeHTTP},
		DeploymentMode: func() *models.McpDeploymentMode {
			mode := models.McpDeploymentModeLocal
			return &mode
		}(),
		SecurityIndicators: &models.McpSecurityIndicator{
			VerifiedSource: &trueVal,
			ReadOnlyTools:  &trueVal,
		},
	}

	redisMcp := models.McpServer{
		ID:          "8",
		Name:        "Redis MCP Server",
		SourceID:    stringToPointer("organization-mcp-source"),
		Description: stringToPointer("Manage Redis key-value stores, caches and pub/sub channels"),
		Provider:    stringToPointer("Redis Ltd"),
		Version:     stringToPointer("0.8.1"),
		License:     stringToPointer("BSD-3-Clause"),
		LicenseLink: stringToPointer("https://opensource.org/licenses/BSD-3-Clause"),
		Logo:        stringToPointer("https://raw.githubusercontent.com/redis/redis-io/master/public/images/redis-white.png"),
		Tags:        []string{"cache", "database", "messaging"},
		ToolCount:   14,
		Transports:  []models.McpTransportType{models.McpTransportTypeHTTP, models.McpTransportTypeSSE},
		DeploymentMode: func() *models.McpDeploymentMode {
			mode := models.McpDeploymentModeRemote
			return &mode
		}(),
		SecurityIndicators: &models.McpSecurityIndicator{
			VerifiedSource: &trueVal,
			SecureEndpoint: &trueVal,
			SAST:           &falseVal,
		},
	}

	standaloneMcp := models.McpServer{
		ID:          "9",
		Name:        "Standalone MCP Server",
		SourceID:    stringToPointer("standalone-mcp-source"),
		Description: stringToPointer("MCP server with no category, available for use"),
		Provider:    stringToPointer("Independent"),
		Version:     stringToPointer("1.0.0"),
		License:     stringToPointer("MIT"),
		Tags:        []string{"standalone", "general"},
		ToolCount:   5,
		Transports:  []models.McpTransportType{models.McpTransportTypeStdio},
		DeploymentMode: func() *models.McpDeploymentMode {
			mode := models.McpDeploymentModeLocal
			return &mode
		}(),
		SecurityIndicators: &models.McpSecurityIndicator{
			VerifiedSource: &falseVal,
			SecureEndpoint: &trueVal,
		},
	}

	allBases := []models.McpServer{prometheusMcp, kubernetesMcp, grafanaMcp, gitMcp, elasticMcp, dynatraceMcp, postgresMcp, redisMcp, standaloneMcp}

	var all []models.McpServer
	nextID := len(allBases) + 1
	suffixes := []string{"-1", "-2", "-3"}

	for _, base := range allBases {
		for _, suffix := range suffixes {
			s := base
			s.Name = base.Name + suffix
			if suffix != "-1" {
				s.ID = fmt.Sprintf("%d", nextID)
				nextID++
			}
			all = append(all, s)
		}
	}

	return all
}

func GetMcpServerListMock() models.McpServerList {
	allMcpServers := GetMcpServerMocks()

	return models.McpServerList{
		Items:         allMcpServers,
		Size:          int32(len(allMcpServers)),
		PageSize:      int32(10),
		NextPageToken: "",
	}
}

func GetMcpToolWithServerMocks() []models.McpToolWithServer {
	trueVal := true
	falseVal := false

	queryTool := models.McpToolWithServer{
		ServerID: "prometheus-mcp",
		Tool: models.McpTool{
			Name:        "query",
			Description: stringToPointer("Execute PromQL queries against the Prometheus time-series database"),
			AccessType:  models.McpToolAccessTypeReadOnly,
			Parameters: []models.McpToolParameter{
				{
					Name:        "query",
					Type:        "string",
					Description: stringToPointer("PromQL query expression"),
					Required:    true,
				},
				{
					Name:        "time",
					Type:        "string",
					Description: stringToPointer("Evaluation timestamp (ISO format)"),
					Required:    false,
				},
				{
					Name:        "timeout",
					Type:        "string",
					Description: stringToPointer("Evaluation timeout duration"),
					Required:    false,
				},
			},
			Revoked: &falseVal,
		},
	}

	getAlertsTool := models.McpToolWithServer{
		ServerID: "prometheus-mcp",
		Tool: models.McpTool{
			Name:        "get_alerts",
			Description: stringToPointer("Retrieve current problems and incidents"),
			AccessType:  models.McpToolAccessTypeReadOnly,
			Parameters:  []models.McpToolParameter{},
			Revoked:     &falseVal,
		},
	}

	getPodsTool := models.McpToolWithServer{
		ServerID: "kubernetes-mcp",
		Tool: models.McpTool{
			Name:        "get_pods",
			Description: stringToPointer("List pods in a namespace"),
			AccessType:  models.McpToolAccessTypeReadOnly,
			Parameters: []models.McpToolParameter{
				{
					Name:        "namespace",
					Type:        "string",
					Description: stringToPointer("Kubernetes namespace"),
					Required:    true,
				},
				{
					Name:        "label_selector",
					Type:        "string",
					Description: stringToPointer("Label selector to filter pods"),
					Required:    false,
				},
			},
			Revoked: &falseVal,
		},
	}

	createMaintenanceWindow := models.McpToolWithServer{
		ServerID: "prometheus-mcp",
		Tool: models.McpTool{
			Name:        "create_maintenance_window",
			Description: stringToPointer("Create a maintenance window to suppress alerts"),
			AccessType:  models.McpToolAccessTypeReadWrite,
			Parameters: []models.McpToolParameter{
				{
					Name:        "name",
					Type:        "string",
					Description: stringToPointer("Maintenance window name"),
					Required:    true,
				},
				{
					Name:        "start_time",
					Type:        "string",
					Description: stringToPointer("Start time (ISO format)"),
					Required:    true,
				},
				{
					Name:        "end_time",
					Type:        "string",
					Description: stringToPointer("End time (ISO format)"),
					Required:    true,
				},
				{
					Name:        "entity_ids",
					Type:        "array",
					Description: stringToPointer("Entity IDs to include in maintenance"),
					Required:    false,
				},
				{
					Name:        "description",
					Type:        "string",
					Description: stringToPointer("Maintenance window description"),
					Required:    false,
				},
			},
			Revoked: &falseVal,
		},
	}

	executeDql := models.McpToolWithServer{
		ServerID: "prometheus-mcp",
		Tool: models.McpTool{
			Name:        "execute_dql",
			Description: stringToPointer("Execute Dynatrace Query Language (DQL) queries"),
			AccessType:  models.McpToolAccessTypeReadOnly,
			Parameters: []models.McpToolParameter{
				{
					Name:        "query",
					Type:        "string",
					Description: stringToPointer("DQL query string"),
					Required:    true,
				},
				{
					Name:        "max_results",
					Type:        "integer",
					Description: stringToPointer("Maximum number of results to return"),
					Required:    false,
				},
			},
			Revoked: &falseVal,
		},
	}

	getServiceHealth := models.McpToolWithServer{
		ServerID: "prometheus-mcp",
		Tool: models.McpTool{
			Name:        "get_service_health",
			Description: stringToPointer("Get health status of services"),
			AccessType:  models.McpToolAccessTypeReadOnly,
			Parameters: []models.McpToolParameter{
				{
					Name:        "service_id",
					Type:        "string",
					Description: stringToPointer("Service identifier"),
					Required:    false,
				},
			},
			Revoked: &falseVal,
		},
	}

	getVulnerabilities := models.McpToolWithServer{
		ServerID: "prometheus-mcp",
		Tool: models.McpTool{
			Name:        "get_vulnerabilities",
			Description: stringToPointer("Retrieve security vulnerability data"),
			AccessType:  models.McpToolAccessTypeReadOnly,
			Parameters: []models.McpToolParameter{
				{
					Name:        "severity",
					Type:        "string",
					Description: stringToPointer("Filter by severity level (critical, high, medium, low)"),
					Required:    false,
				},
				{
					Name:        "status",
					Type:        "string",
					Description: stringToPointer("Filter by vulnerability status (open, resolved, suppressed)"),
					Required:    false,
				},
			},
			Revoked: &falseVal,
		},
	}

	deployModel := models.McpToolWithServer{
		ServerID: "kubernetes-mcp",
		Tool: models.McpTool{
			Name:        "deploy_model",
			Description: stringToPointer("Deploy a machine learning model to a Kubernetes cluster"),
			AccessType:  models.McpToolAccessTypeExecute,
			Parameters: []models.McpToolParameter{
				{
					Name:        "model_name",
					Type:        "string",
					Description: stringToPointer("Name of the model to deploy"),
					Required:    true,
				},
				{
					Name:        "namespace",
					Type:        "string",
					Description: stringToPointer("Target Kubernetes namespace"),
					Required:    true,
				},
				{
					Name:        "replicas",
					Type:        "integer",
					Description: stringToPointer("Number of replicas"),
					Required:    false,
				},
				{
					Name:        "image",
					Type:        "string",
					Description: stringToPointer("Container image URI for the model server"),
					Required:    true,
				},
			},
			Revoked: &falseVal,
		},
	}

	queryRange := models.McpToolWithServer{
		ServerID: "prometheus-mcp",
		Tool: models.McpTool{
			Name:        "query_range",
			Description: stringToPointer("Execute a PromQL range query over a time window"),
			AccessType:  models.McpToolAccessTypeReadOnly,
			Parameters: []models.McpToolParameter{
				{
					Name:        "query",
					Type:        "string",
					Description: stringToPointer("PromQL query expression"),
					Required:    true,
				},
				{
					Name:        "start",
					Type:        "string",
					Description: stringToPointer("Start timestamp (ISO format)"),
					Required:    true,
				},
				{
					Name:        "end",
					Type:        "string",
					Description: stringToPointer("End timestamp (ISO format)"),
					Required:    true,
				},
				{
					Name:        "step",
					Type:        "string",
					Description: stringToPointer("Query resolution step width (e.g. 15s, 1m, 5m)"),
					Required:    false,
				},
			},
			Revoked: &falseVal,
		},
	}

	getMetricMetadata := models.McpToolWithServer{
		ServerID: "prometheus-mcp",
		Tool: models.McpTool{
			Name:        "get_metric_metadata",
			Description: stringToPointer("Retrieve metadata about a specific Prometheus metric"),
			AccessType:  models.McpToolAccessTypeReadOnly,
			Parameters: []models.McpToolParameter{
				{
					Name:        "metric",
					Type:        "string",
					Description: stringToPointer("Metric name"),
					Required:    true,
				},
			},
			Revoked: &falseVal,
		},
	}

	listTargets := models.McpToolWithServer{
		ServerID: "prometheus-mcp",
		Tool: models.McpTool{
			Name:        "list_targets",
			Description: stringToPointer("List all active and dropped scrape targets"),
			AccessType:  models.McpToolAccessTypeReadOnly,
			Parameters: []models.McpToolParameter{
				{
					Name:        "state",
					Type:        "string",
					Description: stringToPointer("Filter by target state (active, dropped, any)"),
					Required:    false,
				},
			},
			Revoked: &falseVal,
		},
	}

	deleteAlertSilence := models.McpToolWithServer{
		ServerID: "prometheus-mcp",
		Tool: models.McpTool{
			Name:        "delete_alert_silence",
			Description: stringToPointer("Delete an alert silence by ID"),
			AccessType:  models.McpToolAccessTypeExecute,
			Parameters: []models.McpToolParameter{
				{
					Name:        "silence_id",
					Type:        "string",
					Description: stringToPointer("ID of the silence to delete"),
					Required:    true,
				},
			},
			Revoked: &falseVal,
		},
	}

	legacyExport := models.McpToolWithServer{
		ServerID: "prometheus-mcp",
		Tool: models.McpTool{
			Name:          "legacy_export",
			Description:   stringToPointer("Export metrics in legacy format (deprecated)"),
			AccessType:    models.McpToolAccessTypeReadOnly,
			Parameters:    []models.McpToolParameter{},
			Revoked:       &trueVal,
			RevokedReason: stringToPointer("This tool has been deprecated in favor of the new metrics API."),
		},
	}

	return []models.McpToolWithServer{
		createMaintenanceWindow,
		executeDql,
		getAlertsTool,
		getServiceHealth,
		getVulnerabilities,
		queryTool,
		queryRange,
		getMetricMetadata,
		listTargets,
		deleteAlertSilence,
		getPodsTool,
		deployModel,
		legacyExport,
	}
}

func GetMcpToolListMock() models.McpToolList {
	tools := GetMcpToolWithServerMocks()

	return models.McpToolList{
		Items:         tools,
		NextPageToken: "",
		PageSize:      25,
		Size:          int32(len(tools)),
	}
}

func GetMcpFilterOptionMocks() map[string]models.FilterOption {
	mcpFilters := make(map[string]models.FilterOption)

	mcpFilters["provider"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"Anthropic",
			"CNCF",
			"Dynatrace",
			"Elastic",
			"Google",
			"OpenAI",
			"Prometheus Community",
			"Red Hat",
		},
	}

	mcpFilters["license"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"Apache 2.0",
			"Elastic 2.0",
			"MIT",
		},
	}

	mcpFilters["tags"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"ai",
			"alerting",
			"analytics",
			"anthropic",
			"assistants",
			"cloud",
			"containers",
			"kubernetes",
			"metrics",
			"monitoring",
			"observability",
			"openshift",
			"orchestration",
			"remote",
			"search",
		},
	}

	mcpFilters["transports"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"http",
			"sse",
			"stdio",
		},
	}

	mcpFilters["deploymentMode"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"local",
			"remote",
		},
	}

	mcpFilters["securityIndicators"] = models.FilterOption{
		Type: FilterOptionTypeString,
		Values: []interface{}{
			"Verified source",
			"Secure endpoint",
			"SAST",
			"Read only tools",
		},
	}

	return mcpFilters
}

func GetMcpNamedQueriesMocks() map[string]map[string]models.FieldFilter {
	namedQueries := make(map[string]map[string]models.FieldFilter)

	// Secure servers query
	namedQueries["secure_servers"] = map[string]models.FieldFilter{
		"verifiedSource": {
			Operator: "=",
			Value:    true,
		},
		"secureEndpoint": {
			Operator: "=",
			Value:    true,
		},
	}

	// Local only deployment
	namedQueries["local_only"] = map[string]models.FieldFilter{
		"deploymentMode": {
			Operator: "=",
			Value:    "local",
		},
	}

	// Read-only tools
	namedQueries["read_only_servers"] = map[string]models.FieldFilter{
		"readOnlyTools": {
			Operator: "=",
			Value:    true,
		},
	}

	return namedQueries
}

func GetMcpFilterOptionsListMock() models.FilterOptionsList {
	mcpFilters := GetMcpFilterOptionMocks()
	mcpNamedQueries := GetMcpNamedQueriesMocks()

	return models.FilterOptionsList{
		Filters:      &mcpFilters,
		NamedQueries: &mcpNamedQueries,
	}
}

func GetMcpServerCatalogSourceMocks() []models.CatalogSource {
	enabled := true
	disabledBool := false
	availableStatus := "available"

	return []models.CatalogSource{
		{
			Id:      "community-mcp-source",
			Name:    "Community MCP Servers",
			Enabled: &enabled,
			Status:  &availableStatus,
			Labels:  []string{"community_mcp_servers"},
		},
		{
			Id:      "organization-mcp-source",
			Name:    "Organization MCP Servers",
			Enabled: &enabled,
			Status:  &availableStatus,
			Labels:  []string{"organization_mcp_servers"},
		},
		{
			Id:      "standalone-mcp-source",
			Name:    "Other MCP Servers",
			Enabled: &enabled,
			Status:  &availableStatus,
			Labels:  []string{},
		},
		{
			Id:      "disabled-mcp-source",
			Name:    "Disabled MCP source",
			Enabled: &disabledBool,
			Status:  &availableStatus,
			Labels:  []string{"disabled_servers"},
		},
	}
}

func GetMcpServerCatalogSourceListMock() models.CatalogSourceList {
	allSources := GetMcpServerCatalogSourceMocks()

	return models.CatalogSourceList{
		Items:         allSources,
		Size:          int32(len(allSources)),
		PageSize:      int32(10),
		NextPageToken: "",
	}
}

func GetMcpServerCatalogLabelListMock() models.CatalogLabelList {
	communityName := "community_mcp_servers"
	communityDisplay := "Community MCP Servers"
	communityDesc := "Community contributed MCP servers from various sources."

	orgName := "organization_mcp_servers"
	orgDisplay := "Organization MCP Servers"
	orgDesc := "MCP servers provided and maintained by your organization."

	labels := []models.CatalogLabel{
		{
			Name:        &communityName,
			DisplayName: &communityDisplay,
			Description: &communityDesc,
		},
		{
			Name:        &orgName,
			DisplayName: &orgDisplay,
			Description: &orgDesc,
		},
	}

	return models.CatalogLabelList{
		Items:         labels,
		Size:          int32(len(labels)),
		PageSize:      int32(10),
		NextPageToken: "",
	}
}
