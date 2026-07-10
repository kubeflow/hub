import * as React from 'react';
import { DEV_MODE } from '~/app/utilities/const';
import type {
  FormTrackingEventProperties,
  LinkTrackingEventProperties,
  SimpleTrackingEventProperties,
} from './trackingTypes';

export type UserInteractionAPI = {
  trackFormEvent: (eventName: string, properties: FormTrackingEventProperties) => void;
  trackLinkEvent: (eventName: string, properties: LinkTrackingEventProperties) => void;
  trackSimpleEvent: (eventName: string, properties?: SimpleTrackingEventProperties) => void;
  trackPageEvent: () => void;
};

const devLogger: UserInteractionAPI = {
  trackFormEvent: (eventName, properties) => {
    if (DEV_MODE) {
      // eslint-disable-next-line no-console
      console.log(`[UserInteraction] trackFormEvent: ${eventName}`, properties);
    }
  },
  trackLinkEvent: (eventName, properties) => {
    if (DEV_MODE) {
      // eslint-disable-next-line no-console
      console.log(`[UserInteraction] trackLinkEvent: ${eventName}`, properties);
    }
  },
  trackSimpleEvent: (eventName, properties) => {
    if (DEV_MODE) {
      // eslint-disable-next-line no-console
      console.log(`[UserInteraction] trackSimpleEvent: ${eventName}`, properties);
    }
  },
  trackPageEvent: () => {
    if (DEV_MODE) {
      // eslint-disable-next-line no-console
      console.log(`[UserInteraction] trackPageEvent: ${window.location.pathname}`);
    }
  },
};

export const UserInteractionContext = React.createContext<UserInteractionAPI>(devLogger);
