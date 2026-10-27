declare module '*.mjs' {
    export interface PanelStatusResult {
        state: 'available' | 'unavailable' | 'unknown';
        message: string;
        version?: string;
        checkedAt: Date;
    }
    export function loadPanelStatus(
        fetcher?: typeof fetch,
        timeoutMs?: number,
    ): Promise<PanelStatusResult>;
}
