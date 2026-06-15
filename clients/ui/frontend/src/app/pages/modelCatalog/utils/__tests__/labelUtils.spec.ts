import {
  findLabelData,
  getLabelDisplayName,
  getLabelDescription,
  orderLabelsByPriority,
} from '~/app/shared/components/catalog';
import { CatalogLabelList, SourceLabel } from '~/app/shared/types/catalogTypes';
import { OTHER_MCP_SERVERS_DISPLAY_NAME } from '~/app/pages/mcpCatalog/const';

const mockCatalogLabels: CatalogLabelList = {
  items: [
    {
      name: 'Red Hat AI',
      displayName: 'Red Hat AI models',
      description: 'Red Hat AI curated models',
    },
    {
      name: 'Red Hat AI Validated',
      displayName: 'Red Hat AI - Validated models',
      description: 'Benchmarked and validated models',
    },
    {
      name: 'Community',
      displayName: 'Community models',
      description: 'Community contributed models',
    },
    {
      name: null,
      displayName: 'Other models',
      description: 'Uncategorized models',
    },
  ],
  size: 4,
  pageSize: 10,
  nextPageToken: '',
};

describe('Label Utilities', () => {
  describe('findLabelData', () => {
    it('should find label by exact name match', () => {
      const result = findLabelData('Red Hat AI', mockCatalogLabels);
      expect(result).toEqual({
        name: 'Red Hat AI',
        displayName: 'Red Hat AI models',
        description: 'Red Hat AI curated models',
      });
    });

    it('should find label with null name using SourceLabel.other', () => {
      const result = findLabelData(SourceLabel.other, mockCatalogLabels);
      expect(result).toEqual({
        name: null,
        displayName: 'Other models',
        description: 'Uncategorized models',
      });
    });

    it('should return undefined when label is not found', () => {
      const result = findLabelData('Nonexistent Label', mockCatalogLabels);
      expect(result).toBeUndefined();
    });

    it('should return undefined when catalogLabels is null', () => {
      const result = findLabelData('Red Hat AI', null);
      expect(result).toBeUndefined();
    });

    it('should return undefined when sourceLabel is undefined', () => {
      const result = findLabelData(undefined, mockCatalogLabels);
      expect(result).toBeUndefined();
    });

    it('should return undefined when catalogLabels.items is empty', () => {
      const emptyLabels: CatalogLabelList = {
        items: [],
        size: 0,
        pageSize: 10,
        nextPageToken: '',
      };
      const result = findLabelData('Red Hat AI', emptyLabels);
      expect(result).toBeUndefined();
    });
  });

  describe('getLabelDisplayName', () => {
    it('should return displayName from API when available', () => {
      const result = getLabelDisplayName('Red Hat AI', mockCatalogLabels);
      expect(result).toBe('Red Hat AI models');
    });

    it('should return displayName for null label (Other models)', () => {
      const result = getLabelDisplayName(SourceLabel.other, mockCatalogLabels);
      expect(result).toBe('Other models');
    });

    it('should fallback to label + "models" when displayName is not available', () => {
      const result = getLabelDisplayName('Unknown Category', mockCatalogLabels);
      expect(result).toBe('Unknown Category models');
    });

    it('should fallback to "Other models" for SourceLabel.other when not in API', () => {
      const emptyLabels: CatalogLabelList = {
        items: [],
        size: 0,
        pageSize: 10,
        nextPageToken: '',
      };
      const result = getLabelDisplayName(SourceLabel.other, emptyLabels);
      expect(result).toBe('Other models');
    });

    it('should not append "models" if already present in label', () => {
      const result = getLabelDisplayName('Custom models', mockCatalogLabels);
      expect(result).toBe('Custom models');
    });

    it('should use custom otherFallback when provided for SourceLabel.other', () => {
      const emptyLabels: CatalogLabelList = {
        items: [],
        size: 0,
        pageSize: 10,
        nextPageToken: '',
      };
      const result = getLabelDisplayName(
        SourceLabel.other,
        emptyLabels,
        OTHER_MCP_SERVERS_DISPLAY_NAME,
      );
      expect(result).toBe('Other MCP servers');
    });

    it('should return empty string for undefined sourceLabel', () => {
      const result = getLabelDisplayName(undefined, mockCatalogLabels);
      expect(result).toBe('');
    });

    it('should handle null catalogLabels gracefully', () => {
      const result = getLabelDisplayName('Red Hat AI', null);
      expect(result).toBe('Red Hat AI models');
    });
  });

  describe('getLabelDescription', () => {
    it('should return description from API when available', () => {
      const result = getLabelDescription('Red Hat AI', mockCatalogLabels);
      expect(result).toBe('Red Hat AI curated models');
    });

    it('should return description for null label (Other models)', () => {
      const result = getLabelDescription(SourceLabel.other, mockCatalogLabels);
      expect(result).toBe('Uncategorized models');
    });

    it('should return undefined when label is not found', () => {
      const result = getLabelDescription('Nonexistent Label', mockCatalogLabels);
      expect(result).toBeUndefined();
    });

    it('should return undefined when catalogLabels is null', () => {
      const result = getLabelDescription('Red Hat AI', null);
      expect(result).toBeUndefined();
    });

    it('should return undefined when sourceLabel is undefined', () => {
      const result = getLabelDescription(undefined, mockCatalogLabels);
      expect(result).toBeUndefined();
    });
  });

  describe('orderLabelsByPriority', () => {
    const sourceLabels = ['Community', 'Unknown Label', 'Red Hat AI Validated', 'Red Hat AI'];

    it('should order labels according to API response', () => {
      const result = orderLabelsByPriority(sourceLabels, mockCatalogLabels);
      // API order: Red Hat AI, Red Hat AI Validated, Community, null
      // Should be: Red Hat AI, Red Hat AI Validated, Community, Unknown Label
      expect(result).toEqual(['Red Hat AI', 'Red Hat AI Validated', 'Community', 'Unknown Label']);
    });

    it('should put unknown labels at the end', () => {
      const result = orderLabelsByPriority(
        ['Unknown 1', 'Red Hat AI', 'Unknown 2'],
        mockCatalogLabels,
      );
      expect(result).toEqual(['Red Hat AI', 'Unknown 1', 'Unknown 2']);
    });

    it('should return original order when catalogLabels is null', () => {
      const result = orderLabelsByPriority(sourceLabels, null);
      expect(result).toEqual(sourceLabels);
    });

    it('should handle empty sourceLabels array', () => {
      const result = orderLabelsByPriority([], mockCatalogLabels);
      expect(result).toEqual([]);
    });

    it('should skip null label entry when ordering', () => {
      const result = orderLabelsByPriority(['Community', 'Red Hat AI'], mockCatalogLabels);
      // Should not include the null entry in the ordering
      expect(result).toEqual(['Red Hat AI', 'Community']);
    });

    it('should handle all labels being unknown', () => {
      const result = orderLabelsByPriority(['Unknown 1', 'Unknown 2'], mockCatalogLabels);
      expect(result).toEqual(['Unknown 1', 'Unknown 2']);
    });

    it('should handle empty catalogLabels items', () => {
      const emptyLabels: CatalogLabelList = {
        items: [],
        size: 0,
        pageSize: 10,
        nextPageToken: '',
      };
      const result = orderLabelsByPriority(sourceLabels, emptyLabels);
      expect(result).toEqual(sourceLabels);
    });

    it('should preserve order of unknown labels relative to each other', () => {
      const result = orderLabelsByPriority(['Z Label', 'A Label', 'Red Hat AI'], mockCatalogLabels);
      expect(result).toEqual(['Red Hat AI', 'Z Label', 'A Label']);
    });
  });
});
