import type { GenericStaticResponse, RouteHandlerController } from 'cypress/types/net-stubbing';
import type { ModelRegistryKind, RoleBindingKind } from 'mod-arch-shared';
import type { Namespace, UserSettings } from 'mod-arch-core';
import { mockModArchResponse } from 'mod-arch-core';
import type {
  ModelArtifact,
  ModelArtifactList,
  ModelRegistry,
  ModelTransferJob,
  ModelVersion,
  ModelVersionList,
  RegisteredModel,
  RegisteredModelList,
} from '~/app/types';
import type {
  CatalogFilterOptionsList,
  CatalogArtifactList,
  CatalogModel,
  CatalogModelList,
  CatalogSourceList,
} from '~/app/modelCatalogTypes';
import type { McpServerList } from '~/app/mcpServerCatalogTypes';

const MODEL_REGISTRY_API_VERSION = 'v1';
export { MODEL_REGISTRY_API_VERSION };
export const MODEL_CATALOG_API_VERSION = 'v1';

type SuccessErrorResponse = {
  success: boolean;
  error?: string;
};

type ApiResponse<V = SuccessErrorResponse> =
  | V
  | GenericStaticResponse<string, V>
  | RouteHandlerController;

type Replacement<R extends string = string> = Record<R, string | undefined>;
type Query<Q extends string = string> = Record<Q, string>;

type Options = { path?: Replacement; query?: Query; times?: number } | null;

/* eslint-disable @typescript-eslint/no-namespace */
declare global {
  namespace Cypress {
    interface Chainable {
      interceptApi: ((
        type: 'GET /api/:apiVersion/model_registry/:modelRegistryName/registered_models',
        options: { path: { modelRegistryName: string; apiVersion: string } },
        response: ApiResponse<RegisteredModelList>,
      ) => Cypress.Chainable<null>) &
        ((
          type: 'POST /api/:apiVersion/model_registry/:modelRegistryName/registered_models',
          options: { path: { modelRegistryName: string; apiVersion: string } },
          response: ApiResponse<RegisteredModel>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'GET /api/:apiVersion/model_registry/:modelRegistryName/registered_models/:registeredModelId/versions',
          options: {
            path: { modelRegistryName: string; apiVersion: string; registeredModelId: number };
          },
          response: ApiResponse<ModelVersionList>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'GET /api/:apiVersion/model_registry/:modelRegistryName/model_versions',
          options: {
            path: { modelRegistryName: string; apiVersion: string };
          },
          response: ApiResponse<ModelVersionList>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'POST /api/:apiVersion/model_registry/:modelRegistryName/registered_models/:registeredModelId/versions',
          options: {
            path: { modelRegistryName: string; apiVersion: string; registeredModelId: number };
          },
          response: ApiResponse<ModelVersion>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'GET /api/:apiVersion/model_registry/:modelRegistryName/registered_models/:registeredModelId',
          options: {
            path: { modelRegistryName: string; apiVersion: string; registeredModelId: number };
          },
          response: ApiResponse<RegisteredModel>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'PATCH /api/:apiVersion/model_registry/:modelRegistryName/registered_models/:registeredModelId',
          options: {
            path: { modelRegistryName: string; apiVersion: string; registeredModelId: number };
          },
          response: ApiResponse<RegisteredModel>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'GET /api/:apiVersion/model_registry/:modelRegistryName/model_versions/:modelVersionId',
          options: {
            path: { modelRegistryName: string; apiVersion: string; modelVersionId: number };
          },
          response: ApiResponse<ModelVersion>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'GET /api/:apiVersion/model_registry/:modelRegistryName/model_versions/:modelVersionId/artifacts',
          options: {
            path: { modelRegistryName: string; apiVersion: string; modelVersionId: number };
          },
          response: ApiResponse<ModelArtifactList>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'POST /api/:apiVersion/model_registry/:modelRegistryName/model_versions/:modelVersionId/artifacts',
          options: {
            path: { modelRegistryName: string; apiVersion: string; modelVersionId: number };
          },
          response: ApiResponse<ModelArtifact>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'PATCH /api/:apiVersion/model_registry/:modelRegistryName/model_versions/:modelVersionId',
          options: {
            path: { modelRegistryName: string; apiVersion: string; modelVersionId: number };
          },
          response: ApiResponse<ModelVersion | undefined>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'GET /api/:apiVersion/model_registry',
          options: { path: { apiVersion: string } },
          response: ApiResponse<ModelRegistry[]>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'GET /api/:apiVersion/settings/model_registry',
          options: { path: { apiVersion: string } },
          response: ApiResponse<ModelRegistryKind[]>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'GET /api/:apiVersion/user',
          options: { path: { apiVersion: string } },
          response: ApiResponse<UserSettings>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'GET /api/:apiVersion/namespaces',
          options: { path: { apiVersion: string } },
          response: ApiResponse<Namespace[]>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'GET /api/:apiVersion/settings/role_bindings',
          options: { path: { apiVersion: string } },
          response: ApiResponse<RoleBindingKind[]>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'GET /api/:apiVersion/model_catalog/models',
          options: {
            path: { apiVersion: string };
            query: {
              source?: string;
              filterQuery?: string;
              sourceLabel?: string;
              searchKeyword?: string;
            };
          },
          response: ApiResponse<CatalogModelList>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'GET /api/:apiVersion/model_catalog/sources',
          options: { path: { apiVersion: string }; query?: { assetType?: string } },
          response: ApiResponse<CatalogSourceList>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'GET /api/:apiVersion/model_catalog/sources/:sourceId/models/:modelName',
          options: { path: { apiVersion: string; sourceId: string; modelName: string } },
          response: ApiResponse<CatalogModel>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'GET /api/:apiVersion/model_catalog/sources/:sourceId/artifacts/:modelName',
          options: { path: { apiVersion: string; sourceId: string; modelName: string } },
          response: ApiResponse<CatalogArtifactList>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'GET /api/:apiVersion/model_catalog/sources/:sourceId/performance_artifacts/:modelName',
          options: { path: { apiVersion: string; sourceId: string; modelName: string } },
          response: ApiResponse<CatalogArtifactList>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'GET /api/:apiVersion/model_catalog/models/filter_options',
          options: { path: { apiVersion: string }; query: { namespace: string } },
          response: ApiResponse<CatalogFilterOptionsList>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'GET /api/:apiVersion/model_registry/:modelRegistryName/model_transfer_jobs/:modelTransferJobId',
          options: {
            path: { modelRegistryName: string; apiVersion: string; modelTransferJobId: string };
          },
          response: ApiResponse<ModelTransferJob>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'POST /api/:apiVersion/model_registry/:modelRegistryName/model_transfer_jobs',
          options: { path: { modelRegistryName: string; apiVersion: string } },
          response: ApiResponse<ModelTransferJob>,
        ) => Cypress.Chainable<null>) &
        ((
          type: 'GET /api/:apiVersion/mcp_catalog/mcp_servers',
          options: {
            path: { apiVersion: string };
            query?: { sourceLabel?: string };
          },
          response: ApiResponse<McpServerList>,
        ) => Cypress.Chainable<null>);
    }
  }
}

Cypress.Commands.add(
  'interceptApi',
  (type: string, ...args: [Options | null, ApiResponse<unknown>] | [ApiResponse<unknown>]) => {
    if (!type) {
      throw new Error('Invalid type parameter.');
    }
    const options = args.length === 2 ? args[0] : null;
    const response = (args.length === 2 ? args[1] : args[0]) ?? '';

    const pathParts = type.match(/:[a-z][a-zA-Z0-9-_]+/g);
    const [method, staticPathname] = type.split(' ');
    let pathname = staticPathname;
    if (pathParts?.length) {
      if (!options || !options.path) {
        throw new Error(`${type}: missing path replacements`);
      }
      const { path: pathReplacements } = options;
      pathParts.forEach((p) => {
        // remove the starting colun from the regex match
        const part = p.substring(1);
        const replacement = pathReplacements[part];
        if (!replacement) {
          throw new Error(`${type} missing path replacement: ${part}`);
        }
        pathname = pathname.replace(new RegExp(`:${part}\\b`), replacement);
      });
    }
    return cy.intercept(
      {
        method,
        pathname: `/model-registry/${pathname}`,
        query: options?.query,
        ...(options?.times && { times: options.times }),
      },
      mockModArchResponse(response),
    );
  },
);
