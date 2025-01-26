export {};

declare global {
  interface Window {
    RUNTIME_CONFIG: {
      API_BASE_URL: string;
      AIRPSACES_JSON_URL: string;
      PRE_FILTER_GEO_JSON: boolean;
    };
  }
}
