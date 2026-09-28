import { findConflictingSource } from '~/app/pages/skillCatalogSettings/utils/duplicateSourceName';
import type { SkillCatalogSourceConfigList } from '~/app/skillCatalogTypes';
import { SkillCatalogSourceType } from '~/app/skillCatalogTypes';

const configs: SkillCatalogSourceConfigList = {
  catalogs: [
    { id: 'acme_skills', name: 'Acme Skills', type: SkillCatalogSourceType.GIT_SKILLS },
    { id: 'community_skills', name: 'Community Skills', type: SkillCatalogSourceType.GIT_SKILLS },
  ],
};

describe('findConflictingSource', () => {
  it('finds the source a same-named new source would collide with', () => {
    expect(findConflictingSource('Acme Skills', configs)?.id).toBe('acme_skills');
  });

  it('matches on the derived id, not the raw name', () => {
    // generateSourceIdFromName lowercases and folds spaces/hyphens to underscores, so
    // these all derive acme_skills and collide even though the names differ.
    expect(findConflictingSource('acme skills', configs)?.id).toBe('acme_skills');
    expect(findConflictingSource('Acme-Skills', configs)?.id).toBe('acme_skills');
  });

  it('does not treat a name as colliding when punctuation changes the derived id', () => {
    // Punctuation is stripped rather than folded, so "Acme's Skills" derives
    // acmes_skills and is a genuinely different source from acme_skills.
    expect(findConflictingSource("Acme's Skills", configs)).toBeUndefined();
  });

  it('returns undefined for a name that derives a free id', () => {
    expect(findConflictingSource('Nvidia Skills', configs)).toBeUndefined();
  });

  it('returns undefined for an empty or punctuation-only name', () => {
    expect(findConflictingSource('', configs)).toBeUndefined();
    expect(findConflictingSource('   ', configs)).toBeUndefined();
    expect(findConflictingSource('!!!', configs)).toBeUndefined();
  });

  it('returns undefined before the source list has loaded', () => {
    expect(findConflictingSource('Acme Skills', null)).toBeUndefined();
  });
});
