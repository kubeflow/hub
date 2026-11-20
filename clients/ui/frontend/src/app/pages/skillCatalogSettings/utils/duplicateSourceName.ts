import { generateSourceIdFromName } from '~/app/shared/catalogSettings';
import type {
  SkillCatalogSourceConfig,
  SkillCatalogSourceConfigList,
} from '~/app/skillCatalogTypes';

/**
 * Returns the existing source whose id a new source of this name would collide with,
 * or undefined when there is no clash.
 *
 * A source id is derived from its name and is not editable, so two sources named alike
 * resolve to the same id. The BFF rejects that on create, but only after submit — and
 * that id is also the key the source's git token is stored under in the shared
 * git-credentials Secret. Detecting it while the admin is still typing avoids both the
 * wasted round trip and the confusion of a failure that looks like it changed nothing.
 *
 * Only meaningful when creating: an edit keeps the id the source already has.
 */
export const findConflictingSource = (
  name: string,
  sourceConfigs: SkillCatalogSourceConfigList | null,
): SkillCatalogSourceConfig | undefined => {
  if (!name.trim()) {
    return undefined;
  }
  const candidateId = generateSourceIdFromName(name);
  if (!candidateId) {
    return undefined;
  }
  return sourceConfigs?.catalogs.find((c) => c.id === candidateId);
};
