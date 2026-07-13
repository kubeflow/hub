/**
 * Vendor-neutral tracking event types.
 *
 * These mirror the main tracking methods available in downstream implementations
 * (e.g., Segment) without introducing any vendor-specific dependency upstream.
 *
 * Event naming convention: "What-noun Past-Tensed-verb"
 * Examples: "Model Deployed", "Pipeline Schedule Deleted", "Workbench Created"
 */

export const enum TrackingOutcome {
  submit = 'submit',
  cancel = 'cancel',
}

export type FormTrackingEventProperties = {
  outcome: TrackingOutcome;
  success?: boolean;
  error?: string;
  [key: string]: string | number | boolean | undefined;
};

export type LinkTrackingEventProperties = {
  /** Destination URL or route path. */
  href?: string;
  /** Kind of element clicked (e.g., "project", "model", "external"). */
  type?: string;
  /** UI region where the click occurred (e.g., "Model Catalog", "Sidebar"). */
  section?: string;
  /** Display label of the clicked element. */
  name?: string;
  [key: string]: string | number | boolean | undefined;
};

export type SimpleTrackingEventProperties = {
  [key: string]: string | number | boolean | undefined;
};
