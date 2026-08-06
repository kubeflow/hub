import * as React from 'react';

export const FORM_LABELS = {
  NAME: 'Name',
  SOURCE_TYPE: 'Source type',
  ORGANIZATION: 'Organization',
  ACCESS_TOKEN: 'Access token',
  YAML_CONTENT: 'Upload a YAML file',
  MODEL_VISIBILITY: 'Model visibility',
  ALLOWED_MODELS: 'Included models',
  EXCLUDED_MODELS: 'Excluded models',
  ENABLE_SOURCE: 'Enable source',
  CREDENTIALS: 'Credentials',
} as const;

export const BUTTON_LABELS = {
  ADD: 'Add',
  SAVE: 'Save',
  PREVIEW: 'Preview',
  CANCEL: 'Cancel',
} as const;

export const SOURCE_TYPE_LABELS = {
  HUGGING_FACE: 'Hugging Face repository',
  YAML: 'YAML file',
} as const;

export { SOURCE_NAME_CHARACTER_LIMIT } from '~/app/shared/catalogSettings/const';

export const VALIDATION_MESSAGES = {
  NAME_REQUIRED: 'Name is required',
  ORGANIZATION_REQUIRED: 'Organization is required',
  YAML_CONTENT_REQUIRED: 'YAML content is required',
} as const;

export const DESCRIPTION_TEXT = {
  ACCESS_TOKEN:
    'Enter your fine-grained Hugging Face access token. The token must have the following permissions: read repos in your namespace, read public repos that you can access, access webhooks, and create webhooks.',
  ORGANIZATION:
    'Enter the name of the organization (for example, meta-llama) to sync models from. Hugging Face sources are limited to 1 organization to prevent performance issues related to loading large model sets.',
  ENABLE_SOURCE:
    'Enable users in your organization to view models from this source in the model catalog.',
  FILTER_INFO_GENERIC:
    'Optionally filter which models from this source appear in the model catalog. If no filters are set, all models from the source will be visible.',
} as const;

export const HELPER_TEXT = {
  ACCESS_TOKEN: 'Enter your Hugging Face access token.',
  YAML: 'Upload or paste a YAML string.',
  ORGANIZATION_SLUG:
    'Hugging Face organization’s name is case-sensitive and should match the organization’s URL, which may differ from the displayed name. Use the organization’s URL slug found in the URL (e.g., Input: meta-llama from huggingface.co/meta-llama).',
} as const;

export const PLACEHOLDERS = {
  ORGANIZATION: 'Example: meta-llama',
  ALLOWED_MODELS: 'Example: Llama*, Llama-3.1-8B-Instruct',
  EXCLUDED_MODELS: 'Example: Llama*, Llama-3.1-8B-Instruct',
} as const;

export const EXPECTED_YAML_FORMAT_LABEL = 'View expected file format';

export const PAGE_TITLES = {
  MODEL_CATALOG_PREVIEW: 'Model catalog preview',
  PREVIEW_MODELS: 'Preview models',
} as const;

export const ERROR_MESSAGES = {
  PREVIEW_FAILED: 'Preview failed',
  SAVE_FAILED: 'Failed to save source',
  FILE_UPLOAD_FAILED: 'File upload failed',
  FILE_UPLOAD_FAILED_BODY:
    "The YAML file couldn't be uploaded. Check its syntax and structure, then try again.",
  VALIDATION_FAILED: 'Validation failed',
  VALIDATION_FAILED_BODY: 'The system cannot establish a connection to the source.',
} as const;

export const SUCCESS_MESSAGES = {
  VALIDATION_SUCCESSFUL: 'Validation successful',
  VALIDATION_SUCCESSFUL_BODY: 'The organization and access token are valid for connection.',
} as const;

export const TABLE_COLUMN_LABELS = {
  SOURCE_NAME: 'Source name',
  ORGANIZATION: 'Organization',
  MODEL_VISIBILITY: 'Model visibility',
  SOURCE_TYPE: 'Source type',
  ENABLE: 'Enable',
  VALIDATION_STATUS: 'Validation status',
} as const;

export const TABLE_COLUMN_POPOVERS = {
  ORGANIZATION:
    'Applies only to Hugging Face sources. Shows the organization the source syncs models from (for example, meta-llama). Only models within this organization are included in the catalog.',
  ENABLE:
    'Enable a source to make its models available to users in your organization from the model catalog.',
} as const;

export const EMPTY_STATE_TEXT = {
  NO_MODELS_INCLUDED: 'No models included',
  NO_MODELS_INCLUDED_BODY:
    'No models from this source are visible in the model catalog. To include models, edit the model visibility settings of this source.',
  NO_MODELS_EXCLUDED: 'No models excluded',
  NO_MODELS_EXCLUDED_BODY: 'No models from this source are excluded by this filter',
} as const;

export const getFilterInfoWithOrg = (organization: string): React.ReactNode => (
  <>
    Optionally filter which <strong>{organization}</strong> models from this source appear in the
    model catalog. If no filters are set, all <strong>{organization}</strong> models from the source
    will be visible.
  </>
);

export const getAllowedModelsHelp = (organization?: string): React.ReactNode =>
  organization ? (
    <>
      Enter the names of <strong>{organization}</strong> models to include from this source. These
      models will appear in the model catalog.
    </>
  ) : (
    'Enter the names of models to include from this source. These models will appear in the model catalog.'
  );

export const getExcludedModelsHelp = (organization?: string): React.ReactNode =>
  organization ? (
    <>
      Enter the names of <strong>{organization}</strong> models to exclude from this source. These
      models will not appear in the model catalog.
    </>
  ) : (
    'Enter the names of models to exclude from this source. These models will not appear in the model catalog.'
  );

/** Same for HF and YAML sources. */
export const getIncludedModelsFieldHelperText =
  'Separate model names using commas. To include all models with a specific prefix, enter the prefix followed by an asterisk. Example, Llama*';

/** Same for HF and YAML sources. */
export const getExcludedModelsFieldHelperText =
  'Separate model names using commas. To exclude all models with a specific prefix, enter the prefix followed by an asterisk. Example, Llama*';
