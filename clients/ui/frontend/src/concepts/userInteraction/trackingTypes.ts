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
  from?: string;
  href?: string;
  to?: string;
  type?: string;
  section?: string;
  name?: string;
};

export type SimpleTrackingEventProperties = {
  [key: string]: string | number | boolean | undefined;
};
