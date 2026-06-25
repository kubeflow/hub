package openapi

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kubeflow/hub/catalog/internal/catalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog"
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	mrmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// timeToMillisStringPointer converts time.Time to *string representing milliseconds since epoch.
func timeToMillisStringPointer(t time.Time) *string {
	s := strconv.FormatInt(t.UnixMilli(), 10)
	return &s
}

// pointerOrDefault returns the value pointed to by p, or def if p is nil.
func pointerOrDefault(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}

func TestFindModels(t *testing.T) {
	// Define common models for testing
	time1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	time2 := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	time3 := time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC)
	time4 := time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC)

	// Updated model definitions to match OpenAPI schema (no direct Id or Published, use Name, CreateTime, LastUpdateTime)
	modelA := &model.CatalogModel{Name: "Model A", CreateTimeSinceEpoch: timeToMillisStringPointer(time1), LastUpdateTimeSinceEpoch: timeToMillisStringPointer(time4)}
	modelB := &model.CatalogModel{Name: "Model B", CreateTimeSinceEpoch: timeToMillisStringPointer(time2), LastUpdateTimeSinceEpoch: timeToMillisStringPointer(time3)}
	modelC := &model.CatalogModel{Name: "Another Model C", CreateTimeSinceEpoch: timeToMillisStringPointer(time3), LastUpdateTimeSinceEpoch: timeToMillisStringPointer(time2)}
	modelD := &model.CatalogModel{Name: "My Model D", CreateTimeSinceEpoch: timeToMillisStringPointer(time4), LastUpdateTimeSinceEpoch: timeToMillisStringPointer(time1)}

	testCases := []struct {
		name              string
		sourceID          string
		mockModels        map[string]*model.CatalogModel
		filterQuery       string
		q                 string
		pageSize          string
		orderBy           model.OrderByField
		sortOrder         model.SortOrder
		nextPageToken     string
		expectedStatus    int
		expectedModelList *model.CatalogModelList
	}{
		{
			name:     "Successful query with no filters",
			sourceID: "source1",
			mockModels: map[string]*model.CatalogModel{
				"modelA": modelA, "modelB": modelB, "modelC": modelC, "modelD": modelD,
			},
			q:              "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_NAME,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedModelList: &model.CatalogModelList{
				Items:         []model.CatalogModel{*modelC, *modelA, *modelB, *modelD}, // Sorted by Name ASC: Another Model C, Model A, Model B, My Model D
				Size:          4,
				PageSize:      10, // Default page size
				NextPageToken: "",
			},
		},
		{
			name:     "Filter by query 'Model'",
			sourceID: "source1",
			mockModels: map[string]*model.CatalogModel{
				"modelA": modelA, "modelB": modelB, "modelC": modelC, "modelD": modelD,
			},
			q:              "Model",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_NAME,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedModelList: &model.CatalogModelList{
				Items:         []model.CatalogModel{*modelC, *modelA, *modelB, *modelD}, // Corrected to include modelC and sorted by name ASC
				Size:          4,                                                        // Corrected from 3 to 4
				PageSize:      10,
				NextPageToken: "",
			},
		},
		{
			name:     "Filter by query 'model' (case insensitive)",
			sourceID: "source1",
			mockModels: map[string]*model.CatalogModel{
				"modelA": modelA, "modelB": modelB, "modelC": modelC, "modelD": modelD,
			},
			q:              "model",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_NAME,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedModelList: &model.CatalogModelList{
				Items:         []model.CatalogModel{*modelC, *modelA, *modelB, *modelD}, // Corrected to include modelC and sorted by name ASC
				Size:          4,                                                        // Corrected from 3 to 4
				PageSize:      10,
				NextPageToken: "",
			},
		},
		{
			name:     "Page size limit",
			sourceID: "source1",
			mockModels: map[string]*model.CatalogModel{
				"modelA": modelA, "modelB": modelB, "modelC": modelC, "modelD": modelD,
			},
			q:              "",
			pageSize:       "2",
			orderBy:        model.ORDERBYFIELD_NAME,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedModelList: &model.CatalogModelList{
				Items:         []model.CatalogModel{*modelC, *modelA}, // First 2 after sorting by Name ASC
				Size:          4,                                      // Total size remains 4
				PageSize:      2,
				NextPageToken: (&stringCursor{Value: "Model A", ID: "Model A"}).String(),
			},
		},
		{
			name:     "Sort by ID Descending (mocked as Name Descending)",
			sourceID: "source1",
			mockModels: map[string]*model.CatalogModel{
				"modelA": modelA, "modelB": modelB, "modelC": modelC, "modelD": modelD,
			},
			q:              "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_DESC,
			expectedStatus: http.StatusOK,
			expectedModelList: &model.CatalogModelList{
				Items:         []model.CatalogModel{*modelD, *modelB, *modelA, *modelC}, // Sorted by Name DESC
				Size:          4,
				PageSize:      10,
				NextPageToken: "",
			},
		},
		{
			name:     "Sort by CreateTime Ascending",
			sourceID: "source1",
			mockModels: map[string]*model.CatalogModel{
				"modelA": modelA, "modelB": modelB, "modelC": modelC, "modelD": modelD,
			},
			q:              "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_CREATE_TIME,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedModelList: &model.CatalogModelList{
				Items:         []model.CatalogModel{*modelA, *modelB, *modelC, *modelD}, // Sorted by CreateTime ASC
				Size:          4,
				PageSize:      10,
				NextPageToken: "",
			},
		},
		{
			name:     "Sort by LastUpdateTime Descending",
			sourceID: "source1",
			mockModels: map[string]*model.CatalogModel{
				"modelA": modelA, "modelB": modelB, "modelC": modelC, "modelD": modelD,
			},
			q:              "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_LAST_UPDATE_TIME,
			sortOrder:      model.SORTORDER_DESC,
			expectedStatus: http.StatusOK,
			expectedModelList: &model.CatalogModelList{
				Items:         []model.CatalogModel{*modelA, *modelB, *modelC, *modelD}, // Corrected to be sorted by LastUpdateTime DESC (modelA has latest time4, modelD has earliest time1)
				Size:          4,
				PageSize:      10,
				NextPageToken: "",
			},
		},
		{
			name:           "Invalid source ID",
			sourceID:       "unknown-source",
			mockModels:     map[string]*model.CatalogModel{},
			q:              "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK, // Changed from http.StatusNotFound to http.StatusOK with an empty list -- now the source ID is just a field in the CatalogModel
			expectedModelList: &model.CatalogModelList{
				Items:         []model.CatalogModel{},
				Size:          0,
				PageSize:      10,
				NextPageToken: "",
			},
		},
		{
			name:     "Invalid pageSize string",
			sourceID: "source1",
			mockModels: map[string]*model.CatalogModel{
				"modelA": modelA,
			},
			q:                 "",
			pageSize:          "abc",
			orderBy:           model.ORDERBYFIELD_ID,
			sortOrder:         model.SORTORDER_ASC,
			expectedStatus:    http.StatusBadRequest,
			expectedModelList: nil,
		},
		{
			name:     "Unsupported orderBy field",
			sourceID: "source1",
			mockModels: map[string]*model.CatalogModel{
				"modelA": modelA,
			},
			q:              "",
			pageSize:       "10",
			orderBy:        "UNSUPPORTED_FIELD",
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK, // Changed from http.StatusBadRequest to http.StatusOK -- in model registry we fallback to ID if the order by field is unsupported
			expectedModelList: &model.CatalogModelList{
				Items: []model.CatalogModel{
					*modelA,
				},
				Size:          1,
				PageSize:      10,
				NextPageToken: "",
			},
		},
		{
			name:     "Unsupported sortOrder field",
			sourceID: "source1",
			mockModels: map[string]*model.CatalogModel{
				"modelA": modelA,
			},
			q:              "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      "UNSUPPORTED_ORDER",
			expectedStatus: http.StatusOK, // Changed from http.StatusBadRequest to http.StatusOK -- in model registry we fallback to ASC if the sort order field is unsupported
			expectedModelList: &model.CatalogModelList{
				Items: []model.CatalogModel{
					*modelA,
				},
				Size:          1,
				PageSize:      10,
				NextPageToken: "",
			},
		},
		{
			name:           "Empty models in source",
			sourceID:       "source1",
			mockModels:     map[string]*model.CatalogModel{},
			q:              "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedModelList: &model.CatalogModelList{
				Items:         []model.CatalogModel{},
				Size:          0,
				PageSize:      10,
				NextPageToken: "",
			},
		},
		{
			name:     "Default sort (ID ascending) and default page size",
			sourceID: "source1",
			mockModels: map[string]*model.CatalogModel{
				"modelB": modelB, "modelA": modelA, "modelD": modelD, "modelC": modelC,
			},
			q:              "",
			pageSize:       "", // Default page size
			orderBy:        "", // Default order by ID
			sortOrder:      "", // Default sort order ASC
			expectedStatus: http.StatusOK,
			expectedModelList: &model.CatalogModelList{
				Items:         []model.CatalogModel{*modelC, *modelA, *modelB, *modelD}, // Sorted by Name ASC (as ID is mocked to use Name)
				Size:          4,
				PageSize:      10, // Default page size
				NextPageToken: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock source collection
			sources := catalog.NewSourceCollection()
			sources.Merge("",
				map[string]catalog.ModelSource{
					"source1": {CatalogSource: model.CatalogSource{Id: "source1", Name: "Test Source 1"}},
				},
			)

			sourceLabels := catalog.NewLabelCollection()

			provider := &mockModelProvider{
				models: tc.mockModels,
			}

			service := NewModelCatalogServiceAPIService(provider, sources, nil, sourceLabels, nil)

			resp, err := service.FindModels(
				context.Background(),
				false, // recommended
				0,     // targetRPS
				"",    // latencyProperty
				"",    // rpsProperty
				"",    // hardwareCountProperty
				"",    // hardwareTypeProperty
				[]string{tc.sourceID},
				tc.q,
				[]string{""},
				tc.filterQuery,
				tc.pageSize,
				tc.orderBy,
				tc.sortOrder,
				tc.nextPageToken,
			)

			assert.Equal(t, tc.expectedStatus, resp.Code)

			if tc.expectedStatus != http.StatusOK {
				assert.Error(t, err)
				return
			}

			require.NotNil(t, resp.Body)
			modelList, ok := resp.Body.(model.CatalogModelList)
			require.True(t, ok, "Response body should be a CatalogModelList")

			assert.Equal(t, tc.expectedModelList.Size, modelList.Size)
			assert.Equal(t, tc.expectedModelList.PageSize, modelList.PageSize)
			if !assert.Equal(t, tc.expectedModelList.NextPageToken, modelList.NextPageToken) && tc.expectedModelList.NextPageToken != "" {
				expectedCursor, _ := decodeStringCursor(tc.expectedModelList.NextPageToken)
				actualCursor, _ := decodeStringCursor(modelList.NextPageToken)
				assert.Equal(t, expectedCursor, actualCursor)
			}

			// Deep equality check for items
			assert.Equal(t, tc.expectedModelList.Items, modelList.Items)
		})
	}
}

func TestFindSources(t *testing.T) {
	// Setup test cases
	trueValue := true
	testCases := []struct {
		name           string
		catalogs       map[string]catalog.ModelSource
		nameFilter     string
		pageSize       string
		orderBy        model.OrderByField
		sortOrder      model.SortOrder
		nextPageToken  string
		expectedStatus int
		expectedSize   int32
		expectedItems  int
		checkSorting   bool
		checkEnabled   bool
		expectedLabels int
	}{
		{
			name:           "Empty catalog list",
			catalogs:       map[string]catalog.ModelSource{},
			nameFilter:     "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedSize:   0,
			expectedItems:  0,
		},
		{
			name: "Single catalog",
			catalogs: map[string]catalog.ModelSource{
				"catalog1": {CatalogSource: model.CatalogSource{Id: "catalog1", Name: "Test Catalog 1", Enabled: &trueValue}},
			},
			nameFilter:     "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedSize:   1,
			expectedItems:  1,
		},
		{
			name: "Multiple catalogs with no filter",
			catalogs: map[string]catalog.ModelSource{
				"catalog1": {CatalogSource: model.CatalogSource{Id: "catalog1", Name: "Test Catalog 1", Enabled: &trueValue}},
				"catalog2": {CatalogSource: model.CatalogSource{Id: "catalog2", Name: "Test Catalog 2", Enabled: &trueValue}},
				"catalog3": {CatalogSource: model.CatalogSource{Id: "catalog3", Name: "Another Catalog", Enabled: &trueValue}},
			},
			nameFilter:     "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedSize:   3,
			expectedItems:  3,
		},
		{
			name: "Filter by name",
			catalogs: map[string]catalog.ModelSource{
				"catalog1": {CatalogSource: model.CatalogSource{Id: "catalog1", Name: "Test Catalog 1", Enabled: &trueValue}},
				"catalog2": {CatalogSource: model.CatalogSource{Id: "catalog2", Name: "Test Catalog 2", Enabled: &trueValue}},
				"catalog3": {CatalogSource: model.CatalogSource{Id: "catalog3", Name: "Another Catalog", Enabled: &trueValue}},
			},
			nameFilter:     "Test",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedSize:   2,
			expectedItems:  2,
		},
		{
			name: "Filter by name case insensitive",
			catalogs: map[string]catalog.ModelSource{
				"catalog1": {CatalogSource: model.CatalogSource{Id: "catalog1", Name: "Test Catalog 1", Enabled: &trueValue}},
				"catalog2": {CatalogSource: model.CatalogSource{Id: "catalog2", Name: "Test Catalog 2", Enabled: &trueValue}},
				"catalog3": {CatalogSource: model.CatalogSource{Id: "catalog3", Name: "Another Catalog", Enabled: &trueValue}},
			},
			nameFilter:     "test",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedSize:   2,
			expectedItems:  2,
		},
		{
			name: "Pagination - limit results",
			catalogs: map[string]catalog.ModelSource{
				"catalog1": {CatalogSource: model.CatalogSource{Id: "catalog1", Name: "Test Catalog 1", Enabled: &trueValue}},
				"catalog2": {CatalogSource: model.CatalogSource{Id: "catalog2", Name: "Test Catalog 2", Enabled: &trueValue}},
				"catalog3": {CatalogSource: model.CatalogSource{Id: "catalog3", Name: "Another Catalog", Enabled: &trueValue}},
			},
			nameFilter:     "",
			pageSize:       "2",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedSize:   2, // Size is the number of items in current page
			expectedItems:  2, // 2 items returned due to page size
		},
		{
			name: "Default page size",
			catalogs: map[string]catalog.ModelSource{
				"catalog1": {CatalogSource: model.CatalogSource{Id: "catalog1", Name: "Test Catalog 1", Enabled: &trueValue}},
				"catalog2": {CatalogSource: model.CatalogSource{Id: "catalog2", Name: "Test Catalog 2", Enabled: &trueValue}},
			},
			nameFilter:     "",
			pageSize:       "", // Empty to test default
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedSize:   2,
			expectedItems:  2,
		},
		{
			name: "Invalid page size",
			catalogs: map[string]catalog.ModelSource{
				"catalog1": {CatalogSource: model.CatalogSource{Id: "catalog1", Name: "Test Catalog 1", Enabled: &trueValue}},
			},
			nameFilter:     "",
			pageSize:       "invalid",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Sort by ID ascending",
			catalogs: map[string]catalog.ModelSource{
				"catalog2": {CatalogSource: model.CatalogSource{Id: "catalog2", Name: "B Catalog", Enabled: &trueValue}},
				"catalog1": {CatalogSource: model.CatalogSource{Id: "catalog1", Name: "A Catalog", Enabled: &trueValue}},
				"catalog3": {CatalogSource: model.CatalogSource{Id: "catalog3", Name: "C Catalog", Enabled: &trueValue}},
			},
			nameFilter:     "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedSize:   3,
			expectedItems:  3,
			checkSorting:   true,
		},
		{
			name: "Sort by ID descending",
			catalogs: map[string]catalog.ModelSource{
				"catalog2": {CatalogSource: model.CatalogSource{Id: "catalog2", Name: "B Catalog", Enabled: &trueValue}},
				"catalog1": {CatalogSource: model.CatalogSource{Id: "catalog1", Name: "A Catalog", Enabled: &trueValue}},
				"catalog3": {CatalogSource: model.CatalogSource{Id: "catalog3", Name: "C Catalog", Enabled: &trueValue}},
			},
			nameFilter:     "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_DESC,
			expectedStatus: http.StatusOK,
			expectedSize:   3,
			expectedItems:  3,
			checkSorting:   true,
		},
		{
			name: "Sort by name ascending",
			catalogs: map[string]catalog.ModelSource{
				"catalog2": {CatalogSource: model.CatalogSource{Id: "catalog2", Name: "B Catalog", Enabled: &trueValue}},
				"catalog1": {CatalogSource: model.CatalogSource{Id: "catalog1", Name: "A Catalog", Enabled: &trueValue}},
				"catalog3": {CatalogSource: model.CatalogSource{Id: "catalog3", Name: "C Catalog", Enabled: &trueValue}},
			},
			nameFilter:     "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_NAME,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedSize:   3,
			expectedItems:  3,
			checkSorting:   true,
		},
		{
			name: "Sort by name descending",
			catalogs: map[string]catalog.ModelSource{
				"catalog2": {CatalogSource: model.CatalogSource{Id: "catalog2", Name: "B Catalog", Enabled: &trueValue}},
				"catalog1": {CatalogSource: model.CatalogSource{Id: "catalog1", Name: "A Catalog", Enabled: &trueValue}},
				"catalog3": {CatalogSource: model.CatalogSource{Id: "catalog3", Name: "C Catalog"}},
			},
			nameFilter:     "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_NAME,
			sortOrder:      model.SORTORDER_DESC,
			expectedStatus: http.StatusOK,
			expectedSize:   3,
			expectedItems:  3,
			checkSorting:   true,
		},
		{
			name: "Invalid sort order",
			catalogs: map[string]catalog.ModelSource{
				"catalog1": {CatalogSource: model.CatalogSource{Id: "catalog1", Name: "Test Catalog 1"}},
			},
			nameFilter:     "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      "INVALID",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid order by field",
			catalogs: map[string]catalog.ModelSource{
				"catalog1": {CatalogSource: model.CatalogSource{Id: "catalog1", Name: "Test Catalog 1"}},
			},
			nameFilter:     "",
			pageSize:       "10",
			orderBy:        "INVALID",
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Default sort order (ID ascending)",
			catalogs: map[string]catalog.ModelSource{
				"catalog2": {CatalogSource: model.CatalogSource{Id: "catalog2", Name: "B Catalog"}},
				"catalog1": {CatalogSource: model.CatalogSource{Id: "catalog1", Name: "A Catalog"}},
				"catalog3": {CatalogSource: model.CatalogSource{Id: "catalog3", Name: "C Catalog"}},
			},
			nameFilter:     "",
			pageSize:       "10",
			orderBy:        "", // Empty to test default
			sortOrder:      "", // Empty to test default
			expectedStatus: http.StatusOK,
			expectedSize:   3,
			expectedItems:  3,
			checkSorting:   true,
		},
		{
			name: "Labels should be returned if set",
			catalogs: map[string]catalog.ModelSource{
				"catalog1": {CatalogSource: model.CatalogSource{Id: "catalog1", Name: "Test Catalog 1", Labels: []string{"label1", "label2"}}},
				"catalog2": {CatalogSource: model.CatalogSource{Id: "catalog2", Name: "Test Catalog 2", Labels: []string{"label3", "label4"}}},
				"catalog3": {CatalogSource: model.CatalogSource{Id: "catalog3", Name: "Test Catalog 3", Labels: []string{"label5", "label6"}}},
			},
			nameFilter:     "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedSize:   3,
			expectedItems:  3,
			checkSorting:   true,
			expectedLabels: 6,
		},
	}

	falseValue := false
	newTestCases := []struct {
		name           string
		catalogs       map[string]catalog.ModelSource
		nameFilter     string
		pageSize       string
		orderBy        model.OrderByField
		sortOrder      model.SortOrder
		nextPageToken  string
		expectedStatus int
		expectedSize   int32
		expectedItems  int
		checkSorting   bool
		checkEnabled   bool
		expectedLabels int
	}{
		{
			name: "All sources returned regardless of enabled status",
			catalogs: map[string]catalog.ModelSource{
				"enabled1":  {CatalogSource: model.CatalogSource{Id: "enabled1", Name: "Enabled Source 1", Enabled: &trueValue}},
				"disabled1": {CatalogSource: model.CatalogSource{Id: "disabled1", Name: "Disabled Source 1", Enabled: &falseValue}},
				"enabled2":  {CatalogSource: model.CatalogSource{Id: "enabled2", Name: "Enabled Source 2", Enabled: &trueValue}},
				"disabled2": {CatalogSource: model.CatalogSource{Id: "disabled2", Name: "Disabled Source 2", Enabled: &falseValue}},
			},
			nameFilter:     "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedSize:   4,
			expectedItems:  4,
			checkEnabled:   true,
		},
		{
			name: "Enabled field present in response for all sources",
			catalogs: map[string]catalog.ModelSource{
				"enabled1":  {CatalogSource: model.CatalogSource{Id: "enabled1", Name: "Enabled Source 1", Enabled: &trueValue}},
				"disabled1": {CatalogSource: model.CatalogSource{Id: "disabled1", Name: "Disabled Source 1", Enabled: &falseValue}},
			},
			nameFilter:     "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedSize:   2,
			expectedItems:  2,
			checkEnabled:   true,
		},
		{
			name: "Name filtering works across all sources (enabled and disabled)",
			catalogs: map[string]catalog.ModelSource{
				"enabled1":  {CatalogSource: model.CatalogSource{Id: "enabled1", Name: "Test Source A", Enabled: &trueValue}},
				"disabled1": {CatalogSource: model.CatalogSource{Id: "disabled1", Name: "Test Source B", Enabled: &falseValue}},
				"enabled2":  {CatalogSource: model.CatalogSource{Id: "enabled2", Name: "Other Source", Enabled: &trueValue}},
				"disabled2": {CatalogSource: model.CatalogSource{Id: "disabled2", Name: "Test Source C", Enabled: &falseValue}},
			},
			nameFilter:     "Test",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedSize:   3, // Should find both enabled and disabled sources with "Test" in name
			expectedItems:  3,
		},
		{
			name: "Pagination accounts for all sources including disabled",
			catalogs: map[string]catalog.ModelSource{
				"enabled1":  {CatalogSource: model.CatalogSource{Id: "enabled1", Name: "Source 1", Enabled: &trueValue}},
				"disabled1": {CatalogSource: model.CatalogSource{Id: "disabled1", Name: "Source 2", Enabled: &falseValue}},
				"enabled2":  {CatalogSource: model.CatalogSource{Id: "enabled2", Name: "Source 3", Enabled: &trueValue}},
				"disabled2": {CatalogSource: model.CatalogSource{Id: "disabled2", Name: "Source 4", Enabled: &falseValue}},
				"enabled3":  {CatalogSource: model.CatalogSource{Id: "enabled3", Name: "Source 5", Enabled: &trueValue}},
			},
			nameFilter:     "",
			pageSize:       "2",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedSize:   2, // Page size, not total
			expectedItems:  2, // First page should have 2 items
		},
		{
			name: "Sorting works across all sources (enabled and disabled interleaved)",
			catalogs: map[string]catalog.ModelSource{
				"source_d": {CatalogSource: model.CatalogSource{Id: "source_d", Name: "D Source", Enabled: &falseValue}},
				"source_b": {CatalogSource: model.CatalogSource{Id: "source_b", Name: "B Source", Enabled: &trueValue}},
				"source_a": {CatalogSource: model.CatalogSource{Id: "source_a", Name: "A Source", Enabled: &falseValue}},
				"source_c": {CatalogSource: model.CatalogSource{Id: "source_c", Name: "C Source", Enabled: &trueValue}},
			},
			nameFilter:     "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedSize:   4,
			expectedItems:  4,
			checkSorting:   true, // Should be sorted a, b, c, d regardless of enabled status
		},
		{
			name:           "Empty catalog returns empty list",
			catalogs:       map[string]catalog.ModelSource{},
			nameFilter:     "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedSize:   0,
			expectedItems:  0,
		},
		{
			name: "All sources disabled still returns all sources",
			catalogs: map[string]catalog.ModelSource{
				"disabled1": {CatalogSource: model.CatalogSource{Id: "disabled1", Name: "Disabled Source 1", Enabled: &falseValue}},
				"disabled2": {CatalogSource: model.CatalogSource{Id: "disabled2", Name: "Disabled Source 2", Enabled: &falseValue}},
				"disabled3": {CatalogSource: model.CatalogSource{Id: "disabled3", Name: "Disabled Source 3", Enabled: &falseValue}},
			},
			nameFilter:     "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_ID,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedSize:   3,
			expectedItems:  3,
			checkEnabled:   true,
		},
		{
			name: "Sorting by NAME works across enabled and disabled sources",
			catalogs: map[string]catalog.ModelSource{
				"source1": {CatalogSource: model.CatalogSource{Id: "source1", Name: "Zebra Catalog", Enabled: &falseValue}},
				"source2": {CatalogSource: model.CatalogSource{Id: "source2", Name: "Alpha Catalog", Enabled: &trueValue}},
				"source3": {CatalogSource: model.CatalogSource{Id: "source3", Name: "Beta Catalog", Enabled: &falseValue}},
				"source4": {CatalogSource: model.CatalogSource{Id: "source4", Name: "Gamma Catalog", Enabled: &trueValue}},
			},
			nameFilter:     "",
			pageSize:       "10",
			orderBy:        model.ORDERBYFIELD_NAME,
			sortOrder:      model.SORTORDER_ASC,
			expectedStatus: http.StatusOK,
			expectedSize:   4,
			expectedItems:  4,
			checkSorting:   true,
		},
	}
	testCases = append(testCases, newTestCases...)

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create service with test catalogs
			sources := catalog.NewSourceCollection()
			sources.Merge("", tc.catalogs)
			sourceLabels := catalog.NewLabelCollection()
			service := NewModelCatalogServiceAPIService(&mockModelProvider{}, sources, nil, sourceLabels, nil)

			// Call FindSources
			resp, err := service.FindSources(
				context.Background(),
				tc.nameFilter,
				"",
				tc.pageSize,
				tc.orderBy,
				tc.sortOrder,
				tc.nextPageToken,
			)

			// Check response status
			assert.Equal(t, tc.expectedStatus, resp.Code)

			// If we expect an error, we don't need to check the response body
			if tc.expectedStatus != http.StatusOK {
				assert.NotNil(t, err)
				return
			}

			// For successful responses, check the response body
			require.NotNil(t, resp.Body)

			// Type assertion to access the CatalogSourceList
			sourceList, ok := resp.Body.(model.CatalogSourceList)
			require.True(t, ok, "Response body should be a CatalogSourceList")

			// Check the size matches expected
			assert.Equal(t, tc.expectedSize, sourceList.Size)

			// Check the number of items matches expected
			assert.Equal(t, tc.expectedItems, len(sourceList.Items))

			// Check that page size is set correctly
			if tc.pageSize == "" {
				// Default page size should be 10
				assert.Equal(t, int32(10), sourceList.PageSize)
			} else if pageSizeInt, err := strconv.ParseInt(tc.pageSize, 10, 32); err == nil {
				assert.Equal(t, int32(pageSizeInt), sourceList.PageSize)
			}

			// Check sorting if required
			if tc.checkSorting && len(sourceList.Items) > 1 {
				switch tc.orderBy {
				case model.ORDERBYFIELD_ID, "":
					// Check ID sorting
					for i := 0; i < len(sourceList.Items)-1; i++ {
						if tc.sortOrder == "" || tc.sortOrder == model.SORTORDER_ASC {
							assert.LessOrEqual(t,
								sourceList.Items[i].Id,
								sourceList.Items[i+1].Id,
								"Items should be sorted by ID in ascending order")
						} else {
							assert.GreaterOrEqual(t,
								sourceList.Items[i].Id,
								sourceList.Items[i+1].Id,
								"Items should be sorted by ID in descending order")
						}
					}
				case model.ORDERBYFIELD_NAME:
					// Check name sorting
					for i := 0; i < len(sourceList.Items)-1; i++ {
						if tc.sortOrder == "" || tc.sortOrder == model.SORTORDER_ASC {
							assert.LessOrEqual(t,
								sourceList.Items[i].Name,
								sourceList.Items[i+1].Name,
								"Items should be sorted by name in ascending order")
						} else {
							assert.GreaterOrEqual(t,
								sourceList.Items[i].Name,
								sourceList.Items[i+1].Name,
								"Items should be sorted by name in descending order")
						}
					}
				}
			}

			labels := make([]string, 0)
			for _, item := range sourceList.Items {
				labels = append(labels, item.Labels...)
			}
			assert.Equal(t, tc.expectedLabels, len(labels))

			// Check enabled field if required
			if tc.checkEnabled {
				for _, item := range sourceList.Items {
					assert.NotNil(t, item.Enabled, "Enabled field should be present for source %s", item.Id)
				}
			}
		})
	}
}

func TestFindLabels(t *testing.T) {
	testCases := []struct {
		name            string
		labels          []map[string]any
		assetType       model.CatalogAssetType
		pageSize        string
		orderBy         string
		sortOrder       model.SortOrder
		nextPageToken   string
		expectedStatus  int
		expectedSize    int32
		expectedItems   int
		checkSorting    bool
		checkOrderByKey string
		expectNextToken bool
	}{
		{
			name:            "Empty labels list",
			labels:          []map[string]any{},
			pageSize:        "10",
			orderBy:         "",
			sortOrder:       model.SORTORDER_ASC,
			expectedStatus:  http.StatusOK,
			expectedSize:    0,
			expectedItems:   0,
			expectNextToken: false,
		},
		{
			name: "Single label",
			labels: []map[string]any{
				{"name": "labelNameOne", "displayName": "Label Name One"},
			},
			pageSize:        "10",
			orderBy:         "",
			sortOrder:       model.SORTORDER_ASC,
			expectedStatus:  http.StatusOK,
			expectedSize:    1,
			expectedItems:   1,
			expectNextToken: false,
		},
		{
			name: "Multiple labels",
			labels: []map[string]any{
				{"name": "labelNameOne", "displayName": "Label Name One"},
				{"name": "community", "displayName": "Community Models"},
				{"name": "enterprise", "displayName": "Enterprise"},
			},
			pageSize:        "10",
			orderBy:         "",
			sortOrder:       model.SORTORDER_ASC,
			expectedStatus:  http.StatusOK,
			expectedSize:    3,
			expectedItems:   3,
			expectNextToken: false,
		},
		{
			name: "Pagination - first page",
			labels: []map[string]any{
				{"name": "label1", "displayName": "Label 1"},
				{"name": "label2", "displayName": "Label 2"},
				{"name": "label3", "displayName": "Label 3"},
				{"name": "label4", "displayName": "Label 4"},
			},
			pageSize:        "2",
			orderBy:         "",
			sortOrder:       model.SORTORDER_ASC,
			expectedStatus:  http.StatusOK,
			expectedSize:    2,
			expectedItems:   2,
			expectNextToken: true,
		},
		{
			name: "Pagination - last page",
			labels: []map[string]any{
				{"name": "label1", "displayName": "Label 1"},
				{"name": "label2", "displayName": "Label 2"},
				{"name": "label3", "displayName": "Label 3"},
			},
			pageSize:        "10",
			orderBy:         "",
			sortOrder:       model.SORTORDER_ASC,
			expectedStatus:  http.StatusOK,
			expectedSize:    3,
			expectedItems:   3,
			expectNextToken: false,
		},
		{
			name: "Sort by name ascending",
			labels: []map[string]any{
				{"name": "zebra", "displayName": "Zebra"},
				{"name": "alpha", "displayName": "Alpha"},
				{"name": "beta", "displayName": "Beta"},
			},
			pageSize:        "10",
			orderBy:         "name",
			sortOrder:       model.SORTORDER_ASC,
			expectedStatus:  http.StatusOK,
			expectedSize:    3,
			expectedItems:   3,
			checkSorting:    true,
			checkOrderByKey: "name",
			expectNextToken: false,
		},
		{
			name: "Sort by name descending",
			labels: []map[string]any{
				{"name": "alpha", "displayName": "Alpha"},
				{"name": "beta", "displayName": "Beta"},
				{"name": "zebra", "displayName": "Zebra"},
			},
			pageSize:        "10",
			orderBy:         "name",
			sortOrder:       model.SORTORDER_DESC,
			expectedStatus:  http.StatusOK,
			expectedSize:    3,
			expectedItems:   3,
			checkSorting:    true,
			checkOrderByKey: "name",
			expectNextToken: false,
		},
		{
			name: "Sort by displayName",
			labels: []map[string]any{
				{"name": "label1", "displayName": "Zebra Display"},
				{"name": "label2", "displayName": "Alpha Display"},
				{"name": "label3", "displayName": "Beta Display"},
			},
			pageSize:        "10",
			orderBy:         "displayName",
			sortOrder:       model.SORTORDER_ASC,
			expectedStatus:  http.StatusOK,
			expectedSize:    3,
			expectedItems:   3,
			checkSorting:    true,
			checkOrderByKey: "displayName",
			expectNextToken: false,
		},
		{
			name: "Labels with missing sort key maintain order",
			labels: []map[string]any{
				{"name": "has-priority", "priority": "high"},
				{"name": "no-priority-1"},
				{"name": "also-has-priority", "priority": "low"},
				{"name": "no-priority-2"},
			},
			pageSize:        "10",
			orderBy:         "priority",
			sortOrder:       model.SORTORDER_ASC,
			expectedStatus:  http.StatusOK,
			expectedSize:    4,
			expectedItems:   4,
			expectNextToken: false,
		},
		{
			name: "Default page size",
			labels: []map[string]any{
				{"name": "label1"},
				{"name": "label2"},
			},
			pageSize:        "",
			orderBy:         "",
			sortOrder:       "",
			expectedStatus:  http.StatusOK,
			expectedSize:    2,
			expectedItems:   2,
			expectNextToken: false,
		},
		{
			name: "Invalid page size",
			labels: []map[string]any{
				{"name": "label1"},
			},
			pageSize:       "invalid",
			orderBy:        "",
			sortOrder:      "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Page size exactly matches items",
			labels: []map[string]any{
				{"name": "label1"},
				{"name": "label2"},
				{"name": "label3"},
			},
			pageSize:        "3",
			orderBy:         "",
			sortOrder:       "",
			expectedStatus:  http.StatusOK,
			expectedSize:    3,
			expectedItems:   3,
			expectNextToken: false,
		},
		{
			name: "Default assetType returns model labels",
			labels: []map[string]any{
				{"name": "model-label", "assetType": "models"},
				{"name": "mcp-label", "assetType": "mcp_servers"},
				{"name": "implicit-model-label"},
			},
			assetType:       "",
			pageSize:        "10",
			expectedStatus:  http.StatusOK,
			expectedSize:    2,
			expectedItems:   2,
			expectNextToken: false,
		},
		{
			name: "Explicit assetType=models",
			labels: []map[string]any{
				{"name": "model-label", "assetType": "models"},
				{"name": "mcp-label", "assetType": "mcp_servers"},
				{"name": "implicit-model-label"},
			},
			assetType:       model.CATALOGASSETTYPE_MODELS,
			pageSize:        "10",
			expectedStatus:  http.StatusOK,
			expectedSize:    2,
			expectedItems:   2,
			expectNextToken: false,
		},
		{
			name: "assetType=mcp_servers",
			labels: []map[string]any{
				{"name": "model-label", "assetType": "models"},
				{"name": "mcp-label-1", "assetType": "mcp_servers"},
				{"name": "mcp-label-2", "assetType": "mcp_servers"},
				{"name": "implicit-model-label"},
			},
			assetType:       model.CATALOGASSETTYPE_MCP_SERVERS,
			pageSize:        "10",
			expectedStatus:  http.StatusOK,
			expectedSize:    2,
			expectedItems:   2,
			expectNextToken: false,
		},
		{
			name: "Invalid assetType returns 400",
			labels: []map[string]any{
				{"name": "label1"},
			},
			assetType:      model.CatalogAssetType("invalid"),
			pageSize:       "10",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Null label with assetType filtering",
			labels: []map[string]any{
				{"name": nil, "displayName": "Community models"},
				{"name": "mcp-label", "displayName": "MCP label", "assetType": "mcp_servers"},
			},
			assetType:       model.CATALOGASSETTYPE_MCP_SERVERS,
			pageSize:        "10",
			expectedStatus:  http.StatusOK,
			expectedSize:    1,
			expectedItems:   1,
			expectNextToken: false,
		},
		{
			name: "Empty result when no labels match assetType",
			labels: []map[string]any{
				{"name": "model-label", "assetType": "models"},
				{"name": "implicit-model-label"},
			},
			assetType:       model.CATALOGASSETTYPE_MCP_SERVERS,
			pageSize:        "10",
			expectedStatus:  http.StatusOK,
			expectedSize:    0,
			expectedItems:   0,
			expectNextToken: false,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create service with test labels
			sources := catalog.NewSourceCollection()
			labelCollection := catalog.NewLabelCollection()
			labelCollection.Merge("test-source", tc.labels)

			service := NewModelCatalogServiceAPIService(&mockModelProvider{}, sources, nil, labelCollection, nil)

			// Call FindLabels
			resp, err := service.FindLabels(
				context.Background(),
				tc.assetType,
				tc.pageSize,
				tc.orderBy,
				tc.sortOrder,
				tc.nextPageToken,
			)

			// Check response status
			assert.Equal(t, tc.expectedStatus, resp.Code)

			// If we expect an error, we don't need to check the response body
			if tc.expectedStatus != http.StatusOK {
				assert.NotNil(t, err)
				return
			}

			// For successful responses, check the response body
			require.NotNil(t, resp.Body)

			// Type assertion to access the CatalogLabelList
			labelList, ok := resp.Body.(model.CatalogLabelList)
			require.True(t, ok, "Response body should be a CatalogLabelList")

			// Check the size matches expected (should be current page size)
			assert.Equal(t, tc.expectedSize, labelList.Size)

			// Check the number of items matches expected
			assert.Equal(t, tc.expectedItems, len(labelList.Items))

			// Check that page size is set correctly
			if tc.pageSize == "" {
				// Default page size should be 10
				assert.Equal(t, int32(10), labelList.PageSize)
			} else if pageSizeInt, err := strconv.ParseInt(tc.pageSize, 10, 32); err == nil {
				assert.Equal(t, int32(pageSizeInt), labelList.PageSize)
			}

			// Check next page token
			if tc.expectNextToken {
				assert.NotEmpty(t, labelList.NextPageToken, "Should have next page token")
			} else {
				assert.Empty(t, labelList.NextPageToken, "Should not have next page token")
			}

			// Check sorting if required
			if tc.checkSorting && len(labelList.Items) > 1 && tc.checkOrderByKey != "" {
				for i := 0; i < len(labelList.Items)-1; i++ {
					// Get value from either Name field or AdditionalProperties
					var val1, val2 *string
					var ok1, ok2 bool

					if tc.checkOrderByKey == "name" {
						val1 = labelList.Items[i].Name.Get()
						val2 = labelList.Items[i+1].Name.Get()
						ok1 = val1 != nil
						ok2 = val2 != nil
					} else {
						var v1, v2 any
						v1, ok1 = labelList.Items[i].AdditionalProperties[tc.checkOrderByKey]
						v2, ok2 = labelList.Items[i+1].AdditionalProperties[tc.checkOrderByKey]
						if ok1 {
							val1, _ = v1.(*string)
							ok1 = val1 != nil
						}
						if ok2 {
							val2, _ = v2.(*string)
							ok2 = val2 != nil
						}
					}

					// Skip if either doesn't have the key
					if !ok1 || !ok2 {
						continue
					}

					if tc.sortOrder == model.SORTORDER_DESC {
						assert.GreaterOrEqual(t,
							*val1,
							*val2,
							"Labels should be sorted by %s in descending order", tc.checkOrderByKey)
					} else {
						assert.LessOrEqual(t,
							*val1,
							*val2,
							"Labels should be sorted by %s in ascending order", tc.checkOrderByKey)
					}
				}
			}
		})
	}
}

// Define a mock model provider
type mockModelProvider struct {
	models              map[string]*model.CatalogModel
	artifacts           map[string][]model.CatalogArtifact
	lastRecommendedSort string // records sortOrder passed to FindModelsWithRecommendedLatency
}

// Mock provider that fails when recommended is used (for testing implementation)
type mockProviderThatFailsOnRecommended struct {
	*mockModelProvider
}

// Implement GetModel method for the mock provider
func (m *mockModelProvider) GetModel(ctx context.Context, name string, sourceID string) (*model.CatalogModel, error) {
	model, exists := m.models[name]
	if !exists {
		return nil, nil
	}
	return model, nil
}

func (m *mockProviderThatFailsOnRecommended) ListModels(ctx context.Context, params catalog.ListModelsParams) (model.CatalogModelList, error) {
	return m.mockModelProvider.ListModels(ctx, params)
}

func (m *mockProviderThatFailsOnRecommended) GetModel(ctx context.Context, name string, sourceID string) (*model.CatalogModel, error) {
	return m.mockModelProvider.GetModel(ctx, name, sourceID)
}

func (m *mockProviderThatFailsOnRecommended) GetArtifacts(ctx context.Context, modelName string, sourceID string, params catalog.ListArtifactsParams) (model.CatalogArtifactList, error) {
	return m.mockModelProvider.GetArtifacts(ctx, modelName, sourceID, params)
}

func (m *mockProviderThatFailsOnRecommended) GetPerformanceArtifacts(ctx context.Context, modelName string, sourceID string, params catalog.ListPerformanceArtifactsParams) (model.CatalogArtifactList, error) {
	return m.mockModelProvider.GetPerformanceArtifacts(ctx, modelName, sourceID, params)
}

func (m *mockProviderThatFailsOnRecommended) GetFilterOptions(ctx context.Context) (*model.FilterOptionsList, error) {
	return m.mockModelProvider.GetFilterOptions(ctx)
}

func (m *mockProviderThatFailsOnRecommended) FindModelsWithRecommendedLatency(ctx context.Context, pagination mrmodels.Pagination, paretoParams modelcatalog.ParetoFilteringParams, sourceIDs []string, query string, sortOrder string) (*model.CatalogModelList, error) {
	return nil, fmt.Errorf("recommended sorting not implemented")
}

func (m *mockModelProvider) ListModels(ctx context.Context, params catalog.ListModelsParams) (model.CatalogModelList, error) {
	var filteredModels []*model.CatalogModel
	for _, mdl := range m.models {
		if params.Query == "" || strings.Contains(strings.ToLower(mdl.Name), strings.ToLower(params.Query)) {
			filteredModels = append(filteredModels, mdl)
		}
	}

	// Sort the filtered models
	sort.SliceStable(filteredModels, func(i, j int) bool {
		cmp := 0
		switch params.OrderBy {
		case model.ORDERBYFIELD_CREATE_TIME:
			// Parse CreateTimeSinceEpoch strings to int64 for comparison
			t1, _ := strconv.ParseInt(pointerOrDefault(filteredModels[i].CreateTimeSinceEpoch, "0"), 10, 64)
			t2, _ := strconv.ParseInt(pointerOrDefault(filteredModels[j].CreateTimeSinceEpoch, "0"), 10, 64)
			cmp = int(t1 - t2)
		case model.ORDERBYFIELD_LAST_UPDATE_TIME:
			// Parse LastUpdateTimeSinceEpoch strings to int64 for comparison
			t1, _ := strconv.ParseInt(pointerOrDefault(filteredModels[i].LastUpdateTimeSinceEpoch, "0"), 10, 64)
			t2, _ := strconv.ParseInt(pointerOrDefault(filteredModels[j].LastUpdateTimeSinceEpoch, "0"), 10, 64)
			cmp = int(t1 - t2)
		case model.ORDERBYFIELD_NAME:
			fallthrough
		default:
			cmp = strings.Compare(filteredModels[i].Name, filteredModels[j].Name)
		}

		if params.SortOrder == model.SORTORDER_DESC {
			return cmp > 0
		}
		return cmp < 0
	})

	totalSize := int32(len(filteredModels))
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	// Apply pagination - limit items to page size
	endIndex := min(int(pageSize), len(filteredModels))

	pagedModels := filteredModels[:endIndex]
	items := make([]model.CatalogModel, len(pagedModels))
	for i, mdl := range pagedModels {
		items[i] = *mdl
	}

	nextPageToken := ""
	if len(filteredModels) > int(pageSize) {
		lastItem := pagedModels[len(pagedModels)-1]
		nextPageToken = (&stringCursor{Value: lastItem.Name, ID: lastItem.Name}).String()
	}

	return model.CatalogModelList{
		Items:         items,
		Size:          totalSize,
		PageSize:      pageSize,
		NextPageToken: nextPageToken,
	}, nil
}

func (m *mockModelProvider) GetArtifacts(ctx context.Context, name string, sourceID string, params catalog.ListArtifactsParams) (model.CatalogArtifactList, error) {
	artifacts, exists := m.artifacts[name]
	if !exists {
		return model.CatalogArtifactList{
			Items:         []model.CatalogArtifact{},
			Size:          0,
			PageSize:      0, // Or a default page size if applicable
			NextPageToken: "",
		}, nil
	}
	return model.CatalogArtifactList{
		Items:         artifacts,
		Size:          int32(len(artifacts)),
		PageSize:      int32(len(artifacts)),
		NextPageToken: "",
	}, nil
}

func (m *mockModelProvider) GetFilterOptions(ctx context.Context) (*model.FilterOptionsList, error) {
	emptyFilters := make(map[string]model.FilterOption)
	return &model.FilterOptionsList{Filters: &emptyFilters}, nil
}

func (m *mockModelProvider) FindModelsWithRecommendedLatency(ctx context.Context, pagination mrmodels.Pagination, paretoParams modelcatalog.ParetoFilteringParams, sourceIDs []string, query string, sortOrder string) (*model.CatalogModelList, error) {
	m.lastRecommendedSort = sortOrder
	// Basic mock implementation - just return models sorted by name
	var allModels []*model.CatalogModel
	for _, mdl := range m.models {
		allModels = append(allModels, mdl)
	}

	// Sort by name for consistent results
	sort.SliceStable(allModels, func(i, j int) bool {
		return allModels[i].Name < allModels[j].Name
	})

	items := make([]model.CatalogModel, len(allModels))
	for i, mdl := range allModels {
		items[i] = *mdl
	}

	return &model.CatalogModelList{
		Items:         items,
		Size:          int32(len(items)),
		PageSize:      10,
		NextPageToken: "",
	}, nil
}

func (m *mockModelProvider) GetPerformanceArtifacts(ctx context.Context, modelName string, sourceID string, params catalog.ListPerformanceArtifactsParams) (model.CatalogArtifactList, error) {
	artifacts, exists := m.artifacts[modelName]
	if !exists {
		return model.CatalogArtifactList{
			Items:         []model.CatalogArtifact{},
			Size:          0,
			PageSize:      params.PageSize,
			NextPageToken: "",
		}, nil
	}

	// Filter for performance artifacts (simplified mock)
	performanceArtifacts := make([]model.CatalogArtifact, 0)
	for _, artifact := range artifacts {
		// In a real implementation, this would check metricsType
		performanceArtifacts = append(performanceArtifacts, artifact)
	}

	// Apply targetRPS calculations if specified
	if params.TargetRPS > 0 {
		for i := range performanceArtifacts {
			if performanceArtifacts[i].CatalogMetricsArtifact != nil {
				if performanceArtifacts[i].CatalogMetricsArtifact.CustomProperties == nil {
					performanceArtifacts[i].CatalogMetricsArtifact.CustomProperties = make(map[string]model.MetadataValue)
				}
				replicas := max(int32(params.TargetRPS/50), 1)
				totalRPS := float64(params.TargetRPS)
				replicasStr := strconv.FormatInt(int64(replicas), 10)
				performanceArtifacts[i].CatalogMetricsArtifact.CustomProperties["replicas"] = model.MetadataIntValueAsMetadataValue(&model.MetadataIntValue{IntValue: replicasStr, MetadataType: "int"})
				performanceArtifacts[i].CatalogMetricsArtifact.CustomProperties["total_requests_per_second"] = model.MetadataDoubleValueAsMetadataValue(&model.MetadataDoubleValue{DoubleValue: totalRPS, MetadataType: "double"})
			}
		}
	}

	return model.CatalogArtifactList{
		Items:         performanceArtifacts,
		Size:          int32(len(performanceArtifacts)),
		PageSize:      params.PageSize,
		NextPageToken: "",
	}, nil
}

func TestGetModel(t *testing.T) {
	testCases := []struct {
		name           string
		sources        map[string]catalog.ModelSource
		sourceID       string
		modelName      string
		expectedStatus int
		expectedModel  *model.CatalogModel
		provider       catalog.APIProvider
	}{
		{
			name: "Existing model in source",
			sources: map[string]catalog.ModelSource{
				"source1": {CatalogSource: model.CatalogSource{Id: "source1", Name: "Test Source"}},
			},
			provider: &mockModelProvider{
				models: map[string]*model.CatalogModel{
					"test-model": {
						Name: "test-model",
					},
				},
			},
			sourceID:       "source1",
			modelName:      "test-model",
			expectedStatus: http.StatusOK,
			expectedModel: &model.CatalogModel{
				Name: "test-model",
			},
		},
		{
			name: "Non-existing source",
			sources: map[string]catalog.ModelSource{
				"source1": {CatalogSource: model.CatalogSource{Id: "source1", Name: "Test Source"}},
			},
			provider: &mockModelProvider{
				models: map[string]*model.CatalogModel{},
			},
			sourceID:       "source2",
			modelName:      "test-model",
			expectedStatus: http.StatusNotFound,
			expectedModel:  nil,
		},
		{
			name: "Existing source, non-existing model",
			sources: map[string]catalog.ModelSource{
				"source1": {CatalogSource: model.CatalogSource{Id: "source1", Name: "Test Source"}},
			},
			provider: &mockModelProvider{
				models: map[string]*model.CatalogModel{},
			},
			sourceID:       "source1",
			modelName:      "test-model",
			expectedStatus: http.StatusNotFound,
			expectedModel:  nil,
		},
		{
			name: "Model name with an escaped slash and version",
			sources: map[string]catalog.ModelSource{
				"source1": {CatalogSource: model.CatalogSource{Id: "source1", Name: "Test Source"}},
			},
			provider: &mockModelProvider{
				models: map[string]*model.CatalogModel{
					"some/model:v1.0.0": {
						Name: "some/model:v1.0.0",
					},
				},
			},
			sourceID:       "source1",
			modelName:      "some%2Fmodel%3Av1.0.0",
			expectedStatus: http.StatusOK,
			expectedModel: &model.CatalogModel{
				Name: "some/model:v1.0.0",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create service with test sources
			sources := catalog.NewSourceCollection()
			sources.Merge("", tc.sources)
			sourceLabels := catalog.NewLabelCollection()
			service := NewModelCatalogServiceAPIService(tc.provider, sources, nil, sourceLabels, nil)

			// Call GetModel
			resp, _ := service.GetModel(
				context.Background(),
				tc.sourceID,
				tc.modelName,
			)

			// Check response status
			assert.Equal(t, tc.expectedStatus, resp.Code)

			// If we expect an error or not found, we don't need to check the response body
			if tc.expectedStatus != http.StatusOK {
				return
			}

			// For successful responses, check the response body
			require.NotNil(t, resp.Body)

			// Type assertion to access the Model
			model, ok := resp.Body.(*model.CatalogModel)
			require.True(t, ok, "Response body should be a Model")

			// Check the model details
			assert.Equal(t, tc.expectedModel.Name, model.Name)
		})
	}
}

func TestGetAllModelArtifacts(t *testing.T) {
	testCases := []struct {
		name              string
		sources           map[string]catalog.ModelSource
		sourceID          string
		modelName         string
		expectedStatus    int
		expectedArtifacts []model.CatalogArtifact
		provider          catalog.APIProvider
	}{
		{
			name: "Existing artifacts for model in source",
			sources: map[string]catalog.ModelSource{
				"source1": {CatalogSource: model.CatalogSource{Id: "source1", Name: "Test Source"}},
			},
			provider: &mockModelProvider{
				artifacts: map[string][]model.CatalogArtifact{
					"test-model": {
						{
							CatalogModelArtifact: &model.CatalogModelArtifact{
								Uri: "s3://bucket/artifact1",
							},
						},
						{
							CatalogModelArtifact: &model.CatalogModelArtifact{
								Uri: "s3://bucket/artifact2",
							},
						},
					},
				},
			},
			sourceID:       "source1",
			modelName:      "test-model",
			expectedStatus: http.StatusOK,
			expectedArtifacts: []model.CatalogArtifact{
				{
					CatalogModelArtifact: &model.CatalogModelArtifact{
						Uri: "s3://bucket/artifact1",
					},
				},
				{
					CatalogModelArtifact: &model.CatalogModelArtifact{
						Uri: "s3://bucket/artifact2",
					},
				},
			},
		},
		{
			name: "Non-existing source",
			sources: map[string]catalog.ModelSource{
				"source1": {CatalogSource: model.CatalogSource{Id: "source1", Name: "Test Source"}},
			},
			provider: &mockModelProvider{
				artifacts: map[string][]model.CatalogArtifact{},
			},
			sourceID:          "source2",
			modelName:         "test-model",
			expectedStatus:    http.StatusOK, // Changed from http.StatusNotFound to http.StatusOK -- having the same behavior as the model registry
			expectedArtifacts: []model.CatalogArtifact{},
		},
		{
			name: "Existing source, no artifacts for model",
			sources: map[string]catalog.ModelSource{
				"source1": {CatalogSource: model.CatalogSource{Id: "source1", Name: "Test Source"}},
			},
			provider: &mockModelProvider{
				artifacts: map[string][]model.CatalogArtifact{},
			},
			sourceID:          "source1",
			modelName:         "test-model",
			expectedStatus:    http.StatusOK,
			expectedArtifacts: []model.CatalogArtifact{}, // Should be an empty slice, not nil
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create service with test sources
			sources := catalog.NewSourceCollection()
			sources.Merge("", tc.sources)
			sourceLabels := catalog.NewLabelCollection()
			service := NewModelCatalogServiceAPIService(tc.provider, sources, nil, sourceLabels, nil)

			// Call GetAllModelArtifacts
			resp, _ := service.GetAllModelArtifacts(
				context.Background(),
				tc.sourceID,
				tc.modelName,
				[]model.ArtifactTypeQueryParam{},
				[]model.ArtifactTypeQueryParam{},
				"",
				"10",
				string(model.ORDERBYFIELD_CREATE_TIME),
				model.SORTORDER_ASC,
				"",
			)

			// Check response status
			assert.Equal(t, tc.expectedStatus, resp.Code)

			// If we expect an error or not found, we don't need to check the response body
			if tc.expectedStatus != http.StatusOK {
				return
			}

			// For successful responses, check the response body
			require.NotNil(t, resp.Body)

			// Type assertion to access the list of artifacts
			artifactList, ok := resp.Body.(model.CatalogArtifactList)
			require.True(t, ok, "Response body should be a CatalogArtifactList")

			// Check the artifacts
			assert.Equal(t, tc.expectedArtifacts, artifactList.Items)
			assert.Equal(t, int32(len(tc.expectedArtifacts)), artifactList.Size)
		})
	}
}

func TestFindModelsFilterOptions(t *testing.T) {
	testCases := []struct {
		name           string
		provider       catalog.APIProvider
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "Successfully retrieve filter options",
			provider: &mockModelProvider{
				models: map[string]*model.CatalogModel{},
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sources := catalog.NewSourceCollection()
			sourceLabels := catalog.NewLabelCollection()
			service := NewModelCatalogServiceAPIService(tc.provider, sources, nil, sourceLabels, nil)

			resp, err := service.FindModelsFilterOptions(context.Background())

			assert.Equal(t, tc.expectedStatus, resp.Code)

			if tc.expectedError {
				assert.Error(t, err)
				return
			}
			require.NotNil(t, resp.Body)

			// Type assertion to access the FilterOptionsList
			filterOptions, ok := resp.Body.(*model.FilterOptionsList)
			require.True(t, ok, "Response body should be a FilterOptionsList")

			require.NotNil(t, filterOptions.Filters)
		})
	}
}

func TestGetAllModelPerformanceArtifacts(t *testing.T) {
	// Define test artifacts
	artifact1Name := "performance-artifact-1"
	artifact2Name := "performance-artifact-2"

	artifact1 := model.CatalogArtifact{
		CatalogMetricsArtifact: &model.CatalogMetricsArtifact{
			Name:             &artifact1Name,
			ArtifactType:     "metrics-artifact",
			MetricsType:      "performance-metrics",
			CustomProperties: map[string]model.MetadataValue{},
		},
	}

	artifact2 := model.CatalogArtifact{
		CatalogMetricsArtifact: &model.CatalogMetricsArtifact{
			Name:             &artifact2Name,
			ArtifactType:     "metrics-artifact",
			MetricsType:      "performance-metrics",
			CustomProperties: map[string]model.MetadataValue{},
		},
	}

	testCases := []struct {
		name              string
		sourceID          string
		modelName         string
		targetRPS         int32
		recommendataions  bool
		filterQuery       string
		pageSize          string
		orderBy           string
		sortOrder         model.SortOrder
		nextPageToken     string
		provider          catalog.APIProvider
		expectedStatus    int
		expectedArtifacts []model.CatalogArtifact
		checkCustomProps  bool
	}{
		{
			name:             "Basic performance artifacts retrieval",
			sourceID:         "source-1",
			modelName:        "test-model",
			targetRPS:        0,
			recommendataions: false,
			filterQuery:      "",
			pageSize:         "10",
			orderBy:          "",
			sortOrder:        model.SORTORDER_ASC,
			nextPageToken:    "",
			provider: &mockModelProvider{
				models: map[string]*model.CatalogModel{
					"test-model": {Name: "test-model"},
				},
				artifacts: map[string][]model.CatalogArtifact{
					"test-model": {artifact1, artifact2},
				},
			},
			expectedStatus:    http.StatusOK,
			expectedArtifacts: []model.CatalogArtifact{artifact1, artifact2},
			checkCustomProps:  false,
		},
		{
			name:             "Performance artifacts with targetRPS parameter",
			sourceID:         "source-1",
			modelName:        "test-model",
			targetRPS:        100,
			recommendataions: false,
			filterQuery:      "",
			pageSize:         "10",
			orderBy:          "",
			sortOrder:        model.SORTORDER_ASC,
			nextPageToken:    "",
			provider: &mockModelProvider{
				models: map[string]*model.CatalogModel{
					"test-model": {Name: "test-model"},
				},
				artifacts: map[string][]model.CatalogArtifact{
					"test-model": {artifact1},
				},
			},
			expectedStatus:    http.StatusOK,
			expectedArtifacts: []model.CatalogArtifact{artifact1},
			checkCustomProps:  true,
		},
		{
			name:             "Performance artifacts with recommendataions enabled",
			sourceID:         "source-1",
			modelName:        "test-model",
			targetRPS:        200,
			recommendataions: true,
			filterQuery:      "",
			pageSize:         "5",
			orderBy:          "",
			sortOrder:        model.SORTORDER_DESC,
			nextPageToken:    "",
			provider: &mockModelProvider{
				models: map[string]*model.CatalogModel{
					"test-model": {Name: "test-model"},
				},
				artifacts: map[string][]model.CatalogArtifact{
					"test-model": {artifact1},
				},
			},
			expectedStatus:    http.StatusOK,
			expectedArtifacts: []model.CatalogArtifact{artifact1},
			checkCustomProps:  true,
		},
		{
			name:             "Model not found",
			sourceID:         "source-1",
			modelName:        "nonexistent-model",
			targetRPS:        0,
			recommendataions: false,
			filterQuery:      "",
			pageSize:         "10",
			orderBy:          "",
			sortOrder:        model.SORTORDER_ASC,
			nextPageToken:    "",
			provider: &mockModelProvider{
				models:    map[string]*model.CatalogModel{},
				artifacts: map[string][]model.CatalogArtifact{},
			},
			expectedStatus:    http.StatusOK,
			expectedArtifacts: []model.CatalogArtifact{},
			checkCustomProps:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sources := catalog.NewSourceCollection()
			sources.Merge("", map[string]catalog.ModelSource{
				tc.sourceID: {
					CatalogSource: model.CatalogSource{Id: tc.sourceID, Name: "Test Source"},
				},
			})
			sourceLabels := catalog.NewLabelCollection()

			service := NewModelCatalogServiceAPIService(tc.provider, sources, nil, sourceLabels, nil)

			resp, err := service.GetAllModelPerformanceArtifacts(
				context.Background(),
				tc.sourceID,
				tc.modelName,
				tc.targetRPS,
				tc.recommendataions,
				"", // rpsProperty
				"", // latencyProperty
				"", // hardwareCountProperty
				"", // hardwareTypeProperty
				tc.filterQuery,
				tc.pageSize,
				tc.orderBy,
				tc.sortOrder,
				tc.nextPageToken,
			)

			require.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, resp.Code)

			if tc.expectedStatus != http.StatusOK {
				return
			}

			// For successful responses, check the response body
			require.NotNil(t, resp.Body)

			// Type assertion to access the list of artifacts
			artifactList, ok := resp.Body.(model.CatalogArtifactList)
			require.True(t, ok, "Response body should be a CatalogArtifactList")

			// Check artifact count
			assert.Equal(t, int32(len(tc.expectedArtifacts)), artifactList.Size)

			// If we need to check custom properties (for targetRPS tests)
			if tc.checkCustomProps && tc.targetRPS > 0 {
				require.Greater(t, len(artifactList.Items), 0, "Should have at least one artifact")

				// Check that custom properties include replicas and total_requests_per_second
				artifact := artifactList.Items[0]
				require.NotNil(t, artifact.CatalogMetricsArtifact, "Should be a metrics artifact")
				require.NotNil(t, artifact.CatalogMetricsArtifact.CustomProperties, "Should have custom properties")

				_, foundReplicas := artifact.CatalogMetricsArtifact.CustomProperties["replicas"]
				_, foundTotalRPS := artifact.CatalogMetricsArtifact.CustomProperties["total_requests_per_second"]

				assert.True(t, foundReplicas, "Should have replicas custom property")
				assert.True(t, foundTotalRPS, "Should have total_requests_per_second custom property")
			}
		})
	}
}

func TestFindModelsRecommended(t *testing.T) {
	// Setup test server and data
	sources := catalog.NewSourceCollection()
	sources.Merge("",
		map[string]catalog.ModelSource{
			"source1": {CatalogSource: model.CatalogSource{Id: "source1", Name: "Test Source 1"}},
		},
	)

	sourceLabels := catalog.NewLabelCollection()

	provider := &mockModelProvider{
		models: map[string]*model.CatalogModel{
			"modelA": {Name: "Model A"},
			"modelB": {Name: "Model B"},
		},
	}

	service := NewModelCatalogServiceAPIService(provider, sources, nil, sourceLabels, nil)

	// Test recommended=true with default parameters
	resp, err := service.FindModels(
		context.Background(),
		true, // recommended
		0,    // targetRPS
		"",   // latencyProperty
		"",   // rpsProperty
		"",   // hardwareCountProperty
		"",   // hardwareTypeProperty
		[]string{"source1"},
		"",
		[]string{""},
		"",
		"10",
		model.ORDERBYFIELD_NAME,
		model.SORTORDER_ASC,
		"",
	)

	assert.Equal(t, http.StatusOK, resp.Code)
	require.NoError(t, err)

	require.NotNil(t, resp.Body)
	response, ok := resp.Body.(model.CatalogModelList)
	require.True(t, ok)

	// Verify models are sorted by recommended latency (mock returns models sorted by name)
	require.True(t, len(response.Items) > 0)
}

func TestFindModelsRecommendedWithCustomParams(t *testing.T) {
	sources := catalog.NewSourceCollection()
	sources.Merge("",
		map[string]catalog.ModelSource{
			"source1": {CatalogSource: model.CatalogSource{Id: "source1", Name: "Test Source 1"}},
		},
	)

	sourceLabels := catalog.NewLabelCollection()

	provider := &mockModelProvider{
		models: map[string]*model.CatalogModel{
			"modelA": {Name: "Model A"},
			"modelB": {Name: "Model B"},
		},
	}

	service := NewModelCatalogServiceAPIService(provider, sources, nil, sourceLabels, nil)

	// Test with custom latency property and targetRPS
	resp, err := service.FindModels(
		context.Background(),
		true,             // recommended
		100,              // targetRPS
		"custom_latency", // latencyProperty
		"",               // rpsProperty
		"",               // hardwareCountProperty
		"",               // hardwareTypeProperty
		[]string{"source1"},
		"",
		[]string{""},
		"",
		"10",
		model.ORDERBYFIELD_NAME,
		model.SORTORDER_ASC,
		"",
	)

	assert.Equal(t, http.StatusOK, resp.Code)
	require.NoError(t, err)

	require.NotNil(t, resp.Body)
	response, ok := resp.Body.(model.CatalogModelList)
	require.True(t, ok)

	// Verify models are sorted by recommended latency (mock returns models sorted by name)
	require.True(t, len(response.Items) > 0)
}

func TestFindModelsRecommendedIgnoresOrderBy(t *testing.T) {
	sources := catalog.NewSourceCollection()
	sources.Merge("",
		map[string]catalog.ModelSource{
			"source1": {CatalogSource: model.CatalogSource{Id: "source1", Name: "Test Source 1"}},
		},
	)

	sourceLabels := catalog.NewLabelCollection()

	provider := &mockModelProvider{
		models: map[string]*model.CatalogModel{
			"modelA": {Name: "Model A"},
			"modelB": {Name: "Model B"},
		},
	}

	service := NewModelCatalogServiceAPIService(provider, sources, nil, sourceLabels, nil)

	// Test that orderBy is ignored when recommended=true
	resp, err := service.FindModels(
		context.Background(),
		true, // recommended
		0,    // targetRPS
		"",   // latencyProperty
		"",   // rpsProperty
		"",   // hardwareCountProperty
		"",   // hardwareTypeProperty
		[]string{"source1"},
		"",
		[]string{""},
		"",
		"10",
		model.ORDERBYFIELD_NAME, // This should be ignored
		model.SORTORDER_ASC,
		"",
	)

	assert.Equal(t, http.StatusOK, resp.Code)
	require.NoError(t, err)

	require.NotNil(t, resp.Body)
	response, ok := resp.Body.(model.CatalogModelList)
	require.True(t, ok)

	// Verify models are sorted by recommended latency (mock returns models sorted by name)
	require.True(t, len(response.Items) > 0)
}

func newTestServiceWithSources(models map[string]*model.CatalogModel) ModelCatalogServiceAPIServicer {
	sources := catalog.NewSourceCollection()
	sources.Merge("",
		map[string]catalog.ModelSource{
			"source1": {CatalogSource: model.CatalogSource{Id: "source1", Name: "Test Source 1"}},
		},
	)

	sourceLabels := catalog.NewLabelCollection()

	provider := &mockModelProvider{
		models: models,
	}

	return NewModelCatalogServiceAPIService(provider, sources, nil, sourceLabels, nil)
}

func TestFindModelsRecommendedWithNumericNextPageToken(t *testing.T) {
	// Regression test: when recommended=true, FindModelsWithRecommendedLatency uses
	// numeric offset tokens (e.g. "10") for pagination. The service must NOT validate
	// these tokens as base64 cursors, which is the format used by DB-backed pagination.
	service := newTestServiceWithSources(map[string]*model.CatalogModel{
		"modelA": {Name: "Model A"},
		"modelB": {Name: "Model B"},
	})

	resp, err := service.FindModels(
		context.Background(),
		true, // recommended
		1,    // targetRPS
		"ttft_p90",
		"",
		"",
		"",
		[]string{"source1"},
		"",
		[]string{""},
		"",
		"10",
		model.ORDERBYFIELD_NAME,
		model.SORTORDER_ASC,
		"10", // numeric nextPageToken from FindModelsWithRecommendedLatency
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestFindModelsNonRecommendedRejectsInvalidToken(t *testing.T) {
	// When recommended=false, invalid nextPageToken should be rejected
	service := newTestServiceWithSources(map[string]*model.CatalogModel{
		"modelA": {Name: "Model A"},
	})

	resp, err := service.FindModels(
		context.Background(),
		false, // NOT recommended
		0,
		"",
		"",
		"",
		"",
		[]string{"source1"},
		"",
		[]string{""},
		"",
		"10",
		model.ORDERBYFIELD_NAME,
		model.SORTORDER_ASC,
		"!!!not-base64", // definitively invalid base64 token
	)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid nextPageToken")
}

func TestFindModelsRecommendedRejectsNonNumericToken(t *testing.T) {
	service := newTestServiceWithSources(map[string]*model.CatalogModel{
		"modelA": {Name: "Model A"},
	})

	resp, err := service.FindModels(
		context.Background(),
		true, // recommended
		1,
		"ttft_p90",
		"",
		"",
		"",
		[]string{"source1"},
		"",
		[]string{""},
		"",
		"10",
		model.ORDERBYFIELD_NAME,
		model.SORTORDER_ASC,
		"abc", // non-numeric token must be rejected
	)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid nextPageToken")
}

func writeTempYAML(t *testing.T, prefix, content string) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), prefix+"-*.yaml")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Sync())
	return f
}

func TestPreviewCatalogSourceAssetTypeRouting(t *testing.T) {
	service := newTestServiceWithSources(map[string]*model.CatalogModel{})

	mcpCatalogData := `
mcp_servers:
  - name: kubernetes-mcp
    description: "Kubernetes server"
  - name: prometheus-mcp
    description: "Prometheus server"
  - name: vault-internal
    description: "Vault internal"
`

	modelCatalogData := `
models:
  - name: "ibm-granite/granite-3b"
    description: "Granite 3B"
  - name: "meta-llama/llama-2-7b"
    description: "Llama 2 7B"
`

	tests := []struct {
		name               string
		config             string
		catalogData        string
		filterStatus       string
		expectedStatus     int
		expectModelResp    bool
		expectedTotalItems int
	}{
		{
			name: "missing assetType defaults to model preview",
			config: `
type: yaml
includedModels:
  - "*"
`,
			catalogData:        modelCatalogData,
			expectedStatus:     http.StatusOK,
			expectModelResp:    true,
			expectedTotalItems: 2,
		},
		{
			name: "explicit assetType models routes to model preview",
			config: `
assetType: models
type: yaml
`,
			catalogData:        modelCatalogData,
			expectedStatus:     http.StatusOK,
			expectModelResp:    true,
			expectedTotalItems: 2,
		},
		{
			name: "assetType mcp_servers routes to MCP preview",
			config: `
assetType: mcp_servers
type: yaml
includedServers:
  - "kubernetes*"
`,
			catalogData:        mcpCatalogData,
			expectedStatus:     http.StatusOK,
			expectModelResp:    false,
			expectedTotalItems: 3,
		},
		{
			name: "unknown assetType falls through to model preview",
			config: `
assetType: agents
type: yaml
`,
			catalogData:        modelCatalogData,
			expectedStatus:     http.StatusOK,
			expectModelResp:    true,
			expectedTotalItems: 2,
		},
		{
			name: "MCP preview with filterStatus included",
			config: `
assetType: mcp_servers
type: yaml
includedServers:
  - "*-mcp"
`,
			catalogData:        mcpCatalogData,
			filterStatus:       "included",
			expectedStatus:     http.StatusOK,
			expectModelResp:    false,
			expectedTotalItems: 2,
		},
		{
			name: "MCP preview with filterStatus excluded",
			config: `
assetType: mcp_servers
type: yaml
includedServers:
  - "*-mcp"
`,
			catalogData:        mcpCatalogData,
			filterStatus:       "excluded",
			expectedStatus:     http.StatusOK,
			expectModelResp:    false,
			expectedTotalItems: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFile := writeTempYAML(t, "config", tt.config)
			var catalogDataFile *os.File
			if tt.catalogData != "" {
				catalogDataFile = writeTempYAML(t, "catalogdata", tt.catalogData)
			}

			resp, err := service.PreviewCatalogSource(
				context.Background(),
				configFile,
				"100",
				"",
				tt.filterStatus,
				catalogDataFile,
			)

			if tt.expectedStatus != http.StatusOK {
				assert.Equal(t, tt.expectedStatus, resp.Code)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.Code)
			require.NotNil(t, resp.Body)

			if tt.expectModelResp {
				modelResp, ok := resp.Body.(model.CatalogSourcePreviewResponse)
				require.True(t, ok, "expected CatalogSourcePreviewResponse, got %T", resp.Body)
				assert.Len(t, modelResp.Items, tt.expectedTotalItems)
				assert.Equal(t, model.CATALOGASSETTYPE_MODELS, modelResp.AssetType)
			} else {
				assetResp, ok := resp.Body.(model.AssetSourcePreviewResponse)
				require.True(t, ok, "expected AssetSourcePreviewResponse, got %T", resp.Body)
				assert.Len(t, assetResp.Items, tt.expectedTotalItems)
				assert.Equal(t, model.CATALOGASSETTYPE_MCP_SERVERS, assetResp.AssetType)
			}
		})
	}
}

func TestPreviewCatalogSourceMCPSummary(t *testing.T) {
	service := newTestServiceWithSources(map[string]*model.CatalogModel{})

	configFile := writeTempYAML(t, "config", `
assetType: mcp_servers
type: yaml
includedServers:
  - "*-mcp"
`)
	catalogDataFile := writeTempYAML(t, "catalogdata", `
mcp_servers:
  - name: kubernetes-mcp
    description: "Kubernetes"
  - name: prometheus-mcp
    description: "Prometheus"
  - name: vault-internal
    description: "Vault"
`)

	resp, err := service.PreviewCatalogSource(
		context.Background(),
		configFile,
		"100",
		"",
		"",
		catalogDataFile,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.Code)

	assetResp, ok := resp.Body.(model.AssetSourcePreviewResponse)
	require.True(t, ok)

	assert.Equal(t, model.CATALOGASSETTYPE_MCP_SERVERS, assetResp.AssetType)
	assert.Equal(t, int32(3), assetResp.Summary.TotalAssets)
	assert.Equal(t, int32(2), assetResp.Summary.IncludedAssets)
	assert.Equal(t, int32(1), assetResp.Summary.ExcludedAssets)
	assert.Len(t, assetResp.Items, 3)
}

func TestPreviewCatalogSourceNilConfig(t *testing.T) {
	service := newTestServiceWithSources(map[string]*model.CatalogModel{})

	resp, err := service.PreviewCatalogSource(
		context.Background(),
		nil,
		"10",
		"",
		"",
		nil,
	)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Error(t, err)
}

func TestFindModelsRecommendedRejectsOverflowToken(t *testing.T) {
	service := newTestServiceWithSources(map[string]*model.CatalogModel{
		"modelA": {Name: "Model A"},
	})

	resp, err := service.FindModels(
		context.Background(),
		true, // recommended
		1,
		"ttft_p90",
		"",
		"",
		"",
		[]string{"source1"},
		"",
		[]string{""},
		"",
		"10",
		model.ORDERBYFIELD_NAME,
		model.SORTORDER_ASC,
		"9999999999999", // exceeds MaxInt32, must be rejected
	)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid nextPageToken")
}

func TestGetAllModelPerformanceArtifactsWithConfigurableProperties(t *testing.T) {
	artifact1Name := "performance-artifact-1"
	artifact1 := model.CatalogArtifact{
		CatalogMetricsArtifact: &model.CatalogMetricsArtifact{
			Name:             &artifact1Name,
			ArtifactType:     "metrics-artifact",
			MetricsType:      "performance-metrics",
			CustomProperties: map[string]model.MetadataValue{},
		},
	}

	testCases := []struct {
		name                  string
		sourceID              string
		modelName             string
		targetRPS             int32
		recommendations       bool
		rpsProperty           string
		latencyProperty       string
		hardwareCountProperty string
		hardwareTypeProperty  string
		provider              catalog.APIProvider
		expectedStatus        int
	}{
		{
			name:                  "Custom property parameters",
			sourceID:              "source1",
			modelName:             "model1",
			targetRPS:             100,
			recommendations:       true,
			rpsProperty:           "throughput",
			latencyProperty:       "p90_latency",
			hardwareCountProperty: "nodes",
			hardwareTypeProperty:  "instance_type",
			provider: &mockModelProvider{
				models: map[string]*model.CatalogModel{
					"model1": {Name: "model1"},
				},
				artifacts: map[string][]model.CatalogArtifact{
					"model1": {artifact1},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:                  "Empty custom property parameters (use defaults)",
			sourceID:              "source1",
			modelName:             "model1",
			targetRPS:             100,
			recommendations:       true,
			rpsProperty:           "",
			latencyProperty:       "",
			hardwareCountProperty: "",
			hardwareTypeProperty:  "",
			provider: &mockModelProvider{
				models: map[string]*model.CatalogModel{
					"model1": {Name: "model1"},
				},
				artifacts: map[string][]model.CatalogArtifact{
					"model1": {artifact1},
				},
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sources := catalog.NewSourceCollection()
			sources.Merge("", map[string]catalog.ModelSource{
				tc.sourceID: {
					CatalogSource: model.CatalogSource{Id: tc.sourceID, Name: "Test Source"},
				},
			})
			sourceLabels := catalog.NewLabelCollection()

			service := NewModelCatalogServiceAPIService(tc.provider, sources, nil, sourceLabels, nil)

			resp, err := service.GetAllModelPerformanceArtifacts(
				context.Background(),
				tc.sourceID,
				tc.modelName,
				tc.targetRPS,
				tc.recommendations,
				tc.rpsProperty,
				tc.latencyProperty,
				tc.hardwareCountProperty,
				tc.hardwareTypeProperty,
				"",                  // filterQuery
				"10",                // pageSize
				"",                  // orderBy
				model.SORTORDER_ASC, // sortOrder
				"")                  // nextPageToken

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, resp.Code)
		})
	}
}

func TestFindModelsOrderByRecommended(t *testing.T) {
	sources := catalog.NewSourceCollection()
	sources.Merge("",
		map[string]catalog.ModelSource{
			"source1": {CatalogSource: model.CatalogSource{Id: "source1", Name: "Test Source 1"}},
		},
	)

	provider := &mockModelProvider{
		models: map[string]*model.CatalogModel{
			"modelA": {Name: "Model A"},
			"modelB": {Name: "Model B"},
		},
	}

	service := NewModelCatalogServiceAPIService(provider, sources, nil, catalog.NewLabelCollection(), nil)

	// orderBy=RECOMMENDED with recommendations=false should still use the recommendation path
	resp, err := service.FindModels(
		context.Background(),
		false, // recommendations=false
		0,     // targetRPS
		"",    // latencyProperty
		"",    // rpsProperty
		"",    // hardwareCountProperty
		"",    // hardwareTypeProperty
		[]string{"source1"},
		"",
		[]string{""},
		"",
		"10",
		model.OrderByField("RECOMMENDED"),
		model.SORTORDER_ASC,
		"",
	)

	assert.Equal(t, http.StatusOK, resp.Code)
	require.NoError(t, err)

	require.NotNil(t, resp.Body)
	response, ok := resp.Body.(model.CatalogModelList)
	require.True(t, ok)
	require.True(t, len(response.Items) > 0)

	// Verify the recommendation path was taken (mock records it)
	assert.Equal(t, "ASC", provider.lastRecommendedSort)
}

func TestFindModelsOrderByRecommendedDesc(t *testing.T) {
	sources := catalog.NewSourceCollection()
	sources.Merge("",
		map[string]catalog.ModelSource{
			"source1": {CatalogSource: model.CatalogSource{Id: "source1", Name: "Test Source 1"}},
		},
	)

	provider := &mockModelProvider{
		models: map[string]*model.CatalogModel{
			"modelA": {Name: "Model A"},
			"modelB": {Name: "Model B"},
		},
	}

	service := NewModelCatalogServiceAPIService(provider, sources, nil, catalog.NewLabelCollection(), nil)

	// orderBy=RECOMMENDED with sortOrder=DESC
	resp, err := service.FindModels(
		context.Background(),
		false, // recommendations=false
		0,     // targetRPS
		"",    // latencyProperty
		"",    // rpsProperty
		"",    // hardwareCountProperty
		"",    // hardwareTypeProperty
		[]string{"source1"},
		"",
		[]string{""},
		"",
		"10",
		model.OrderByField("RECOMMENDED"),
		model.SORTORDER_DESC,
		"",
	)

	assert.Equal(t, http.StatusOK, resp.Code)
	require.NoError(t, err)

	require.NotNil(t, resp.Body)
	_, ok := resp.Body.(model.CatalogModelList)
	require.True(t, ok)

	// Verify DESC was passed to the recommendation path
	assert.Equal(t, "DESC", provider.lastRecommendedSort)
}

// TestFindModelsOrderByRecommendedPagination is a regression test for the case where
// recommendations=false and orderBy=RECOMMENDED: the numeric nextPageToken must not
// be rejected by the base64-cursor validator used by the non-recommended path.
func TestFindModelsOrderByRecommendedPagination(t *testing.T) {
	sources := catalog.NewSourceCollection()
	sources.Merge("",
		map[string]catalog.ModelSource{
			"source1": {CatalogSource: model.CatalogSource{Id: "source1", Name: "Test Source 1"}},
		},
	)

	provider := &mockModelProvider{
		models: map[string]*model.CatalogModel{
			"modelA": {Name: "Model A"},
		},
	}

	service := NewModelCatalogServiceAPIService(provider, sources, nil, catalog.NewLabelCollection(), nil)

	// Pass a numeric nextPageToken (the format produced by the recommended path).
	// Before the fix, this was rejected with 400 because parsePaginationParams tried
	// to base64-decode it as a DB cursor.
	resp, err := service.FindModels(
		context.Background(),
		false, // recommendations=false — using orderBy instead
		0, "", "", "", "",
		[]string{"source1"},
		"", []string{""}, "",
		"10",
		model.OrderByField("RECOMMENDED"),
		model.SORTORDER_ASC,
		"2", // numeric offset token from a prior recommended-path response
	)

	assert.Equal(t, http.StatusOK, resp.Code, "numeric nextPageToken must be accepted for orderBy=RECOMMENDED")
	require.NoError(t, err)
}
