import * as React from 'react';
import useSkillCatalogSettingsAPIState, {
  SkillCatalogSettingsAPIState,
} from '~/app/hooks/skillCatalogSettings/useSkillCatalogSettingsAPIState';
import { useSkillCatalogSourceConfigs } from '~/app/hooks/skillCatalogSettings/useSkillCatalogSourceConfigs';
import type { SkillCatalogSourceConfigList } from '~/app/skillCatalogTypes';
import type { CatalogSourceList } from '~/app/shared/types/catalogTypes';
import { BFF_API_VERSION, URL_PREFIX } from '~/app/utilities/const';
import { createCatalogSettingsContext } from '~/app/shared/catalogSettings/createCatalogSettingsContext';

const { useCatalogSettingsValue } = createCatalogSettingsContext<
  SkillCatalogSettingsAPIState,
  SkillCatalogSourceConfigList
>({
  settingsHostPath: `${URL_PREFIX}/api/${BFF_API_VERSION}/settings/skill_catalog`,
  catalogHostPath: `${URL_PREFIX}/api/${BFF_API_VERSION}/model_catalog`,
  catalogExtraQueryParams: { assetType: 'skills' },
  useSettingsAPIState: useSkillCatalogSettingsAPIState,
  useSourceConfigsList: useSkillCatalogSourceConfigs,
});

export type SkillCatalogSettingsContextType = {
  apiState: SkillCatalogSettingsAPIState;
  refreshAPIState: () => void;
  skillCatalogSourceConfigs: SkillCatalogSourceConfigList | null;
  skillCatalogSourceConfigsLoaded: boolean;
  skillCatalogSourceConfigsLoadError?: Error;
  refreshSkillCatalogSourceConfigs: () => void;
  skillCatalogSources: CatalogSourceList | null;
  skillCatalogSourcesLoaded: boolean;
  skillCatalogSourcesLoadError?: Error;
  refreshSkillCatalogSources: () => void;
  /**
   * True between saving a source and the catalog confirming it reloaded. The
   * status column shows "Syncing" while this holds, because the status the
   * catalog reports until then describes the previous sync, not the edit.
   */
  isSkillSourceSyncPending: (sourceId: string) => boolean;
  /** Call after a successful create/update/toggle so the status column reflects it. */
  markSkillSourceSyncPending: (sourceId: string) => void;
};

type SkillCatalogSettingsContextProviderProps = {
  children: React.ReactNode;
};

export const SkillCatalogSettingsContext = React.createContext<SkillCatalogSettingsContextType>({
  apiState: {
    apiAvailable: false,
    // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
    api: null as unknown as SkillCatalogSettingsAPIState['api'],
  },
  refreshAPIState: () => undefined,
  skillCatalogSourceConfigs: null,
  skillCatalogSourceConfigsLoaded: false,
  skillCatalogSourceConfigsLoadError: undefined,
  refreshSkillCatalogSourceConfigs: () => undefined,
  skillCatalogSources: null,
  skillCatalogSourcesLoaded: false,
  skillCatalogSourcesLoadError: undefined,
  refreshSkillCatalogSources: () => undefined,
  isSkillSourceSyncPending: () => false,
  markSkillSourceSyncPending: () => undefined,
});

export const SkillCatalogSettingsContextProvider: React.FC<
  SkillCatalogSettingsContextProviderProps
> = ({ children }) => {
  const {
    apiState,
    refreshAPIState,
    sourceConfigs: skillCatalogSourceConfigs,
    sourceConfigsLoaded: skillCatalogSourceConfigsLoaded,
    sourceConfigsLoadError: skillCatalogSourceConfigsLoadError,
    refreshSourceConfigs: refreshSkillCatalogSourceConfigs,
    catalogSources: skillCatalogSources,
    catalogSourcesLoaded: skillCatalogSourcesLoaded,
    catalogSourcesLoadError: skillCatalogSourcesLoadError,
    refreshCatalogSources: refreshSkillCatalogSources,
    isSyncPending: isSkillSourceSyncPending,
    markSyncPending: markSkillSourceSyncPending,
  } = useCatalogSettingsValue();

  const contextValue = React.useMemo(
    () => ({
      apiState,
      refreshAPIState,
      skillCatalogSourceConfigs,
      skillCatalogSourceConfigsLoaded,
      skillCatalogSourceConfigsLoadError,
      refreshSkillCatalogSourceConfigs,
      skillCatalogSources,
      skillCatalogSourcesLoaded,
      skillCatalogSourcesLoadError,
      refreshSkillCatalogSources,
      isSkillSourceSyncPending,
      markSkillSourceSyncPending,
    }),
    [
      apiState,
      refreshAPIState,
      skillCatalogSourceConfigs,
      skillCatalogSourceConfigsLoaded,
      skillCatalogSourceConfigsLoadError,
      refreshSkillCatalogSourceConfigs,
      skillCatalogSources,
      skillCatalogSourcesLoaded,
      skillCatalogSourcesLoadError,
      refreshSkillCatalogSources,
      isSkillSourceSyncPending,
      markSkillSourceSyncPending,
    ],
  );

  return (
    <SkillCatalogSettingsContext.Provider value={contextValue}>
      {children}
    </SkillCatalogSettingsContext.Provider>
  );
};
