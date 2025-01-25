export {};

declare global {
  interface Window {
    RUNTIME_CONFIG: {
      API_BASE_URL: string;
      AIRPSACES_JSON_URL: string;
    };
  }
}
