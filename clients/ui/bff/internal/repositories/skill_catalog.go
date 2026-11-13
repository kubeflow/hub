package repositories

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/hub/ui/bff/internal/models"
)

const skillPath = "/skills"
const skillFilterOptionPath = "/skills/filter_options"
const skillMarketplacePath = "/claude/marketplace.json"

type SkillCatalogInterface interface {
	GetAllSkills(client httpclient.HTTPClientInterface, pageValues url.Values) (*models.SkillList, error)
	GetSkillsFilter(client httpclient.HTTPClientInterface) (*models.FilterOptionsList, error)
	GetSkill(client httpclient.HTTPClientInterface, skillId string, pageValues url.Values) (*models.Skill, error)
	GetSkillMarketplace(client httpclient.HTTPClientInterface, pageValues url.Values) (*models.SkillMarketplace, error)
}

type SkillCatalog struct {
	SkillCatalogInterface
}

func (s *SkillCatalog) GetAllSkills(client httpclient.HTTPClientInterface, pageValues url.Values) (*models.SkillList, error) {
	responseData, err := client.GET(UrlWithPageParams(skillPath, pageValues))
	if err != nil {
		return nil, fmt.Errorf("error fetching skills list: %w", err)
	}

	var skills models.SkillList
	if err := json.Unmarshal(responseData, &skills); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &skills, nil
}

func (s *SkillCatalog) GetSkillsFilter(client httpclient.HTTPClientInterface) (*models.FilterOptionsList, error) {
	responseData, err := client.GET(skillFilterOptionPath)
	if err != nil {
		return nil, fmt.Errorf("error fetching skill filter options: %w", err)
	}

	var filters models.FilterOptionsList
	if err := json.Unmarshal(responseData, &filters); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &filters, nil
}

func (s *SkillCatalog) GetSkill(client httpclient.HTTPClientInterface, skillId string, pageValues url.Values) (*models.Skill, error) {
	path, err := url.JoinPath(skillPath, skillId)
	if err != nil {
		return nil, err
	}

	responseData, err := client.GET(UrlWithPageParams(path, pageValues))
	if err != nil {
		return nil, fmt.Errorf("error fetching skill: %w", err)
	}

	var skill models.Skill
	if err := json.Unmarshal(responseData, &skill); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &skill, nil
}

func (s *SkillCatalog) GetSkillMarketplace(client httpclient.HTTPClientInterface, pageValues url.Values) (*models.SkillMarketplace, error) {
	responseData, err := client.GET(UrlWithPageParams(skillMarketplacePath, pageValues))
	if err != nil {
		return nil, fmt.Errorf("error fetching skill marketplace: %w", err)
	}

	var marketplace models.SkillMarketplace
	if err := json.Unmarshal(responseData, &marketplace); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &marketplace, nil
}
