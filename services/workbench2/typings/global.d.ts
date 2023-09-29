declare interface Window {
  __REDUX_DEVTOOLS_EXTENSION__: any;
  __REDUX_DEVTOOLS_EXTENSION_COMPOSE__: any;
}

declare interface NodeModule {
  hot?: { accept: (path: string, callback: () => void) => void };
}

declare interface System {
  import<T = any>(module: string): Promise<T>
}
declare var System: System;

declare module 'react-splitter-layout';
declare module 'react-rte';

declare module 'is-image' {
  export default function isImage(value: string): boolean;
}