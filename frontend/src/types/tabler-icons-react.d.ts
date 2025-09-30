/* eslint-disable @typescript-eslint/no-explicit-any */
/* This file declares Tabler icon components and intentionally allows `any` props
   so icons can accept common props like `size` and `stroke` without lint noise. */

declare module '@tabler/icons-react' {
  import * as React from 'react';
  // Allow any props so icon components accept common props like `size` and `stroke`.
  export const IconCheck: React.ComponentType<any>;
  export const IconX: React.ComponentType<any>;
  export const IconClock: React.ComponentType<any>;
  export const IconPalette: React.ComponentType<any>;
  export const IconUser: React.ComponentType<any>;
  export const IconBrain: React.ComponentType<any>;
  export const IconTarget: React.ComponentType<any>;
  export const IconBell: React.ComponentType<any>;
  export const IconRefresh: React.ComponentType<any>;
  export const IconInfoCircle: React.ComponentType<any>;
  export const IconKeyboard: React.ComponentType<any>;
  export const IconChevronLeft: React.ComponentType<any>;
  export const IconChevronRight: React.ComponentType<any>;
  export const IconSun: React.ComponentType<any>;
  export const IconMoon: React.ComponentType<any>;
  export const IconDeviceDesktop: React.ComponentType<any>;
  export const IconBook: React.ComponentType<any>;
  export const IconVocabulary: React.ComponentType<any>;
  export const IconMessage: React.ComponentType<any>;
  export const IconCalendar: React.ComponentType<any>;
  export const IconLogout: React.ComponentType<any>;
  export const IconBook2: React.ComponentType<any>;
  export const IconAbc: React.ComponentType<any>;
}
