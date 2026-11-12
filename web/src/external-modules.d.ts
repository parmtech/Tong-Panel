declare module '*.mjs' {
    export function mountACMEPreflight(root?: Document): void;
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
    export function loadNginxPreview(siteID: string, fetcher?: typeof fetch): Promise<string>;
    export interface SiteSummary { site_id: string; primary_domain: string; ownership_state: string; }
    export function loadSitesPage(after?: string, fetcher?: typeof fetch, signal?: AbortSignal): Promise<{sites: SiteSummary[]; next_after: string}>;
    export interface FileEntry { name: string; kind: 'file' | 'directory' | 'symlink' | 'other'; size: number; }
    export function listTenantFiles(path: string, fetcher?: typeof fetch): Promise<FileEntry[]>;
    export function downloadTenantFile(path: string, fetcher?: typeof fetch): Promise<{ name: string; blob: Blob }>;
    export interface MyJob {
        id: string;
        tenant_id: string;
        requested_by?: string;
        kind: string;
        state: string;
        idempotency_key: string;
        created_at: string;
        updated_at: string;
    }
    export interface MyJobsOptions { signal?: AbortSignal; timeoutMs?: number; }
    export class MyJobsError extends Error { code: string; }
    export function loadMyJobs(fetcher?: typeof fetch, options?: MyJobsOptions): Promise<MyJob[]>;
    export interface MyJobsPage { jobs: MyJob[]; next_after: string; }
    export function loadMyJobsPage(after?: string, fetcher?: typeof fetch, options?: MyJobsOptions): Promise<MyJobsPage>;
    export function loadMyJob(id: string, fetcher?: typeof fetch, options?: MyJobsOptions): Promise<MyJob>;
}
