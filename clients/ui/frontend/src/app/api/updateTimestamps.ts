import { ModelRegistryAPIs, ModelState, ModelVersion, RegisteredModel } from '~/app/types';

type MinimalModelRegistryAPI = Pick<ModelRegistryAPIs, 'patchRegisteredModel'>;

export const bumpModelVersionTimestamp = async (
  api: ModelRegistryAPIs,
  modelVersion: ModelVersion,
): Promise<void> => {
  if (!modelVersion.id) {
    throw new Error('Model version ID is required');
  }

  try {
    await api.patchModelVersion(
      {},
      {
        state: ModelState.LIVE,
      },
      modelVersion.id,
    );
  } catch (error) {
    throw new Error(
      `Failed to update model version timestamp: ${
        error instanceof Error ? error.message : String(error)
      }`,
    );
  }
};

export const bumpRegisteredModelTimestamp = async (
  api: MinimalModelRegistryAPI,
  registeredModel: RegisteredModel,
): Promise<void> => {
  if (!registeredModel.id) {
    throw new Error('Registered model ID is required');
  }

  try {
    await api.patchRegisteredModel(
      {},
      {
        state: ModelState.LIVE,
      },
      registeredModel.id,
    );
  } catch (error) {
    throw new Error(
      `Failed to update registered model timestamp: ${
        error instanceof Error ? error.message : String(error)
      }`,
    );
  }
};

export const bumpBothTimestamps = async (
  api: ModelRegistryAPIs,
  registeredModel: RegisteredModel,
  modelVersion: ModelVersion,
): Promise<void> => {
  await Promise.all([
    bumpModelVersionTimestamp(api, modelVersion),
    bumpRegisteredModelTimestamp(api, registeredModel),
  ]);
};
