import { kebabTableColumn, SortableData } from 'mod-arch-shared';
import type { SkillCatalogSourceConfig } from '~/app/skillCatalogTypes';

export const skillCatalogSourceConfigsColumns: SortableData<SkillCatalogSourceConfig>[] = [
  {
    field: 'name',
    label: 'Name',
    sortable: (a, b) => a.name.localeCompare(b.name),
    width: 25,
  },
  {
    field: 'type',
    label: 'Source type',
    sortable: false,
    width: 20,
  },
  {
    field: 'enabled',
    label: 'Enable',
    sortable: false,
    info: {
      popover:
        'Enable a source to make its skills available to users in your organization from the skill catalog.',
    },
    width: 15,
  },
  {
    field: 'status',
    label: 'Validation status',
    sortable: false,
    width: 15,
  },
  {
    field: 'actions',
    label: '',
    sortable: false,
    width: 20,
  },
  kebabTableColumn(),
];
