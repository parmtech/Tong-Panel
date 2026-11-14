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
    export interface SiteDatabase { id:string; name:string; username:string; }
    export function loadSiteDatabasePage(siteID:string, after?:string, fetcher?:typeof fetch, signal?:AbortSignal): Promise<{databases:SiteDatabase[];next_after:string}>;
    export function loadSiteDatabases(siteID:string, fetcher?:typeof fetch, signal?:AbortSignal): Promise<SiteDatabase[]>;
    export interface SiteDetails { siteID: string; documentRoot: string; domains: {name:string;kind:string;state:string}[]; }
    export function loadSiteDetails(siteID: string, fetcher?: typeof fetch, signal?: AbortSignal): Promise<SiteDetails>;
    export interface SiteSummary { site_id: string; primary_domain: string; ownership_state: string; }
    export function loadSitesPage(after?: string, fetcher?: typeof fetch, signal?: AbortSignal, prefix?: string): Promise<{sites: SiteSummary[]; next_after: string}>;
    export class FilesExplorerError extends Error { status: number; }
    export interface FileEntry { name: string; kind: 'file' | 'directory' | 'symlink' | 'other'; size: number; }
    export function listTenantFiles(path: string, fetcher?: typeof fetch): Promise<FileEntry[]>;
    export function listTenantFilesPage(path: string, after?: string, fetcher?: typeof fetch, timeoutOverride?: number, prefix?: string): Promise<{entries: FileEntry[]; next_after: string}>;
    export function previewTenantText(path: string, fetcher?: typeof fetch): Promise<string>;
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
