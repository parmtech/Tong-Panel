import { loadPanelStatus } from './panel-status.mjs';
import { loadNginxPreview } from './site-preview.mjs';
import { loadSitesPage } from './sites-list.mjs';
import { mountACMEPreflight } from './acme-preflight.mjs';
import { downloadTenantFile, listTenantFilesPage, previewTenantText, FilesExplorerError, type FileEntry } from './files-explorer.mjs';
import { loadMyJobsPage, loadMyJob, MyJobsError, type MyJob } from './my-jobs.mjs';

export {};
mountACMEPreflight();
type Tile = { title: string; icon: string; subtitle: string; stage: string; searchTerms?: string[] };
const tiles: Tile[] = [
 {title:'Websites',icon:'🌐',subtitle:'Sites & PHP',stage:'P2',searchTerms:['PHP version']},
 {title:'Domains',icon:'🪐',subtitle:'Aliases & redirects',stage:'P2'},
 {title:'Applications',icon:'🧩',subtitle:'Node.js, Python & more',stage:'P3',searchTerms:['new WordPress','Node app']},
 {title:'File Explorer',icon:'📁',subtitle:'Files, uploads & editor',stage:'P2'},
 {title:'Databases',icon:'🗃️',subtitle:'MySQL & PostgreSQL',stage:'P2/P4'},
 {title:'Email',icon:'✉️',subtitle:'Mailboxes & delivery',stage:'P4',searchTerms:['make email']},
 {title:'SSL & HTTPS',icon:'🔒',subtitle:'Certificates & renewals',stage:'P2',searchTerms:['SSL redirect']},
 {title:'DNS',icon:'📡',subtitle:'Records & nameservers',stage:'P4'},
 {title:'Backups',icon:'💾',subtitle:'Back up & restore',stage:'P2'},
 {title:'Server Health',icon:'📊',subtitle:'System monitoring',stage:'P1+'},
 {title:'User Accounts',icon:'👥',subtitle:'Roles & packages',stage:'P1/P4'},
 {title:'Marketplace',icon:'🎁',subtitle:'One-click app recipes',stage:'P3+'},
];
const list = document.querySelector<HTMLElement>('#tiles');
const search = document.querySelector<HTMLInputElement>('#search');
const status = document.querySelector<HTMLElement>('#status');
const statusVersion = document.querySelector<HTMLElement>('#status-version');
const statusChecked = document.querySelector<HTMLElement>('#status-checked');
const refreshStatusButton = document.querySelector<HTMLButtonElement>('#refresh-status');
const theme = document.querySelector<HTMLButtonElement>('#theme');
const intro = document.querySelector<HTMLDialogElement>('#intro');
const sitePreviewForm = document.querySelector<HTMLFormElement>('#site-preview-form');
const sitePreviewID = document.querySelector<HTMLInputElement>('#site-preview-id');
const sitePreviewStatus = document.querySelector<HTMLElement>('#site-preview-status');
const sitePreviewOutput = document.querySelector<HTMLElement>('#site-preview-output');
const sitePreviewSubmit = document.querySelector<HTMLButtonElement>('#site-preview-submit');
let sitePreviewRequest = 0;
const normalizeSearchText = (text: string): string => text.trim().toLowerCase().replace(/\s+/g, ' ');
const filterText = (text: string): void => {
 if (!list) return;
 list.replaceChildren();
 const needle=normalizeSearchText(text);
 const shown=tiles.filter(t=>normalizeSearchText(`${t.title} ${t.subtitle} ${(t.searchTerms ?? []).join(' ')}`).includes(needle));
 for(const t of shown){
   const card=document.createElement('article');card.className='tile';
   const ico=document.createElement('span');ico.className='tile-icon';ico.textContent=t.icon;ico.setAttribute('aria-hidden','true');
   const head=document.createElement('h3');head.textContent=t.title;
   const detail=document.createElement('p');detail.textContent=t.subtitle;
   const tag=document.createElement('span');tag.className='planned';tag.textContent=`Planned · ${t.stage}`;
   card.append(ico,head,detail,tag);list.appendChild(card);
 }
 const counter=document.querySelector<HTMLElement>('#count'); if(counter) counter.textContent=`${shown.length} module${shown.length === 1 ? '' : 's'} planned`;
};
search?.addEventListener('input',()=>filterText(search.value));
filterText('');
theme?.addEventListener('click',()=>{
  const dark=document.documentElement.dataset.theme!=='dark';document.documentElement.dataset.theme=dark?'dark':'light';
  theme.setAttribute('aria-label',dark?'Switch to light mode':'Switch to dark mode');
});
document.querySelector('#tour')?.addEventListener('click',()=>intro?.showModal());
document.querySelector('#close-tour')?.addEventListener('click',()=>intro?.close());
async function refreshStatus():Promise<void>{
 if(!status || !refreshStatusButton)return;
 refreshStatusButton.disabled=true;refreshStatusButton.setAttribute('aria-busy','true');
 status.textContent='Checking panel process availability…';status.dataset.state='unknown';
 try{
  const result=await loadPanelStatus();
  status.textContent=result.message;status.dataset.state=result.state;
  if(statusVersion)statusVersion.textContent=result.version ?? '';
  if(statusChecked)statusChecked.textContent=new Intl.DateTimeFormat(undefined,{hour:'numeric',minute:'2-digit'}).format(result.checkedAt);
 }catch{
  status.textContent='Panel status could not be checked. Retry to check again.';status.dataset.state='unknown';
  if(statusVersion)statusVersion.textContent='';
  if(statusChecked)statusChecked.textContent='—';
 }finally{refreshStatusButton.disabled=false;refreshStatusButton.removeAttribute('aria-busy');}
}
refreshStatusButton?.addEventListener('click',()=>void refreshStatus());
void refreshStatus();
const sitesLoad = document.querySelector<HTMLButtonElement>('#sites-load');
const sitesStatus = document.querySelector<HTMLElement>('#sites-status');
const sitesList = document.querySelector<HTMLUListElement>('#sites-list');
const sitesPager = document.querySelector<HTMLElement>('#sites-pagination');
const sitesPrevious = document.querySelector<HTMLButtonElement>('#sites-previous');
const sitesNext = document.querySelector<HTMLButtonElement>('#sites-next');
const sitesPageLabel = document.querySelector<HTMLElement>('#sites-page');
let sitesCursors = [''], sitesNextAfter = '', sitesRequest = 0;
let sitesController: AbortController | undefined;
async function showSites(cursors: string[], moveFocus = false): Promise<void> {
 if (!sitesStatus || !sitesList || !sitesLoad) return;
 const request = ++sitesRequest; sitesController?.abort(); sitesController = new AbortController();
 const controller = sitesController;
 sitesList.replaceChildren(); sitesList.setAttribute('aria-busy', 'true');
 if (sitesPager) sitesPager.hidden = true;
 sitesStatus.textContent = 'Loading sites…'; sitesStatus.dataset.state = 'loading';
 if (moveFocus) sitesStatus.focus();
 try {
  const page = await loadSitesPage(cursors[cursors.length - 1], undefined, controller.signal);
  if (request !== sitesRequest || controller.signal.aborted) return;
  sitesCursors = cursors; sitesNextAfter = page.next_after;
  for (const site of page.sites) {
   const row = document.createElement('li'); row.className = 'job-entry';
   const label = document.createElement('span'); label.textContent = `${site.site_id} · ${site.primary_domain} · ${site.ownership_state}`;
   const button = document.createElement('button'); button.type = 'button'; button.textContent = 'Use for preview';
   button.setAttribute('aria-label', `Use ${site.site_id} for preview`);
   button.addEventListener('click', () => {
    ++sitePreviewRequest;
    if (sitePreviewOutput) { sitePreviewOutput.hidden = true; sitePreviewOutput.textContent = ''; }
    if (sitePreviewStatus) { sitePreviewStatus.dataset.state = ''; sitePreviewStatus.textContent = 'No preview loaded.'; }
    if (sitePreviewID) { sitePreviewID.value = site.site_id; sitePreviewID.focus(); }
   });
   row.append(label, button); sitesList.append(row);
  }
  sitesStatus.dataset.state = page.sites.length ? 'available' : 'empty';
  sitesStatus.textContent = page.sites.length ? `Page ${cursors.length}: ${page.sites.length} catalog sites. Domain status does not confirm hosting is provisioned.` : 'No sites found in this workspace.';
  if (sitesPager && sitesPrevious && sitesNext && sitesPageLabel) {
   sitesPrevious.disabled = cursors.length === 1; sitesNext.disabled = !page.next_after;
   sitesPageLabel.textContent = `Page ${cursors.length}`; sitesPager.hidden = cursors.length === 1 && !page.next_after;
  }
 } catch (error) {
  if (request !== sitesRequest || controller.signal.aborted) return;
  sitesCursors = ['']; sitesNextAfter = ''; sitesStatus.dataset.state = 'unavailable';
  sitesStatus.textContent = error instanceof Error ? error.message : 'Sites could not be loaded. Try again.';
 } finally { if (request === sitesRequest) sitesList.removeAttribute('aria-busy'); }
}
sitesLoad?.addEventListener('click', () => void showSites(['']));
sitesPrevious?.addEventListener('click', () => { if (sitesCursors.length > 1) void showSites(sitesCursors.slice(0, -1), true); });
sitesNext?.addEventListener('click', () => { if (sitesNextAfter) void showSites([...sitesCursors, sitesNextAfter], true); });
sitePreviewForm?.addEventListener('submit', async (event: SubmitEvent) => {
 event.preventDefault();
 if(!sitePreviewID || !sitePreviewStatus || !sitePreviewOutput || !sitePreviewSubmit)return;
 const request = ++sitePreviewRequest;
 sitePreviewSubmit.disabled=true;sitePreviewSubmit.setAttribute('aria-busy','true');
 sitePreviewStatus.dataset.state='loading';sitePreviewStatus.textContent='Loading read-only preview…';
 sitePreviewOutput.hidden=true;sitePreviewOutput.textContent='';
 try{
  const config=await loadNginxPreview(sitePreviewID.value);
  if(request !== sitePreviewRequest)return;
  sitePreviewOutput.textContent=config;sitePreviewOutput.hidden=false;
  sitePreviewStatus.dataset.state='available';sitePreviewStatus.textContent='Preview loaded. Configuration has not been applied.';
 }catch(error){
  if(request !== sitePreviewRequest)return;
  sitePreviewStatus.dataset.state='unavailable';
  sitePreviewStatus.textContent=error instanceof Error?error.message:'Preview unavailable. Try again.';
 }finally{sitePreviewSubmit.disabled=false;sitePreviewSubmit.removeAttribute('aria-busy');}
});

const jobsRefresh = document.querySelector<HTMLButtonElement>('#jobs-refresh');
const jobsStatus = document.querySelector<HTMLElement>('#jobs-status');
const jobsNotices = document.querySelector<HTMLUListElement>('#jobs-notifications');
const jobsList = document.querySelector<HTMLUListElement>('#jobs-list');
const jobsPagination = document.querySelector<HTMLElement>('#jobs-pagination');
const jobsPrevious = document.querySelector<HTMLButtonElement>('#jobs-previous');
const jobsNext = document.querySelector<HTMLButtonElement>('#jobs-next');
const jobsPage = document.querySelector<HTMLElement>('#jobs-page');
const jobDetail = document.querySelector<HTMLElement>('#job-detail');
const jobDetailTitle = document.querySelector<HTMLElement>('#job-detail-title');
const jobDetailStatus = document.querySelector<HTMLElement>('#job-detail-status');
const jobDetailFields = document.querySelector<HTMLDListElement>('#job-detail-fields');
const jobDetailRetry = document.querySelector<HTMLButtonElement>('#job-detail-retry');
let jobsRequest = 0;
let jobStateScope = '';
let detailRequest = 0;
let jobsController: AbortController | undefined;
let detailController: AbortController | undefined;
let selectedJob: string | undefined;
let jobsCursors: string[] = [''];
let jobsNextAfter = '';
const jobStates = new Map<string, string>();

function renderJobChanges(jobs: MyJob[]): void {
 const scope = jobs.length && jobs.every(job => job.requested_by && job.tenant_id === jobs[0].tenant_id && job.requested_by === jobs[0].requested_by)
  ? `${jobs[0].tenant_id}\u0000${jobs[0].requested_by}` : '';
 if(scope !== jobStateScope){
  jobStates.clear(); jobsNotices?.replaceChildren(); jobStateScope = scope;
 }
 if(!scope) return;
 const changes: {id:string;kind:string;previous:string;current:string}[] = [];
 for(const job of jobs){
  const previous = jobStates.get(job.id);
  if(previous !== undefined && previous !== job.state) changes.push({id:job.id,kind:job.kind,previous,current:job.state});
  if(!jobStates.has(job.id) && jobStates.size >= 5000) jobStates.delete(jobStates.keys().next().value!);
  jobStates.set(job.id,job.state);
 }
 if(!jobsNotices || changes.length === 0) return;
 for(const change of changes){
  const item = document.createElement('li');
  const message = document.createElement('span');
  message.textContent = `Job ${change.id} (${change.kind}) changed from ${change.previous} to ${change.current}.`;
  const dismiss = document.createElement('button'); dismiss.type = 'button'; dismiss.textContent = 'Dismiss';
  dismiss.setAttribute('aria-label', `Dismiss update for job ${change.id}`);
  dismiss.addEventListener('click', () => item.remove());
  item.append(message, dismiss); jobsNotices.prepend(item);
 }
 while(jobsNotices.children.length > 5) jobsNotices.lastElementChild?.remove();
}

function jobsErrorCode(error: unknown): string {
 return error instanceof MyJobsError ? error.code : 'unavailable';
}
function jobsErrorMessage(code: string): string {
 switch(code){
  case 'session_required': return 'A secure browser session is required. Public sign-in is not available in this preview.';
  case 'auth_unconfigured': return 'Job access is not configured on this panel. Public sign-in is not available in this preview.';
  case 'forbidden': return 'Your current session cannot read jobs. Refresh after your access changes.';
  case 'not_found': return 'This job was not found for your current access. Retry or refresh the list.';
  case 'jobs_unconfigured': return 'Job storage is not enabled on this panel. Refresh jobs to retry.';
  case 'jobs_unavailable': return 'Job storage is unavailable. Try again.';
  case 'timeout': return 'The job request timed out. Try again.';
  case 'invalid_response': return 'The panel returned an unsupported job response. Try again.';
  default: return 'The job request could not be completed. Try again.';
 }
}
function isJobAccessFailure(code: string): boolean {
 return code === 'session_required' || code === 'auth_unconfigured' || code === 'forbidden';
}
function clearJobDetail(): void {
 selectedJob = undefined;
 ++detailRequest;
 detailController?.abort();
 jobDetailFields?.replaceChildren();
 if(jobDetail) { jobDetail.hidden = true; jobDetail.removeAttribute('aria-busy'); }
 if(jobDetailStatus) jobDetailStatus.textContent = '';
 if(jobDetailRetry) jobDetailRetry.hidden = true;
}
function clearJobsForAccessFailure(code: string): void {
 ++jobsRequest;
 jobsController?.abort();
 jobsList?.replaceChildren();
 jobsList?.removeAttribute('aria-busy');
 jobsCursors = ['']; jobsNextAfter = '';
 if(jobsPagination) jobsPagination.hidden = true;
 clearJobDetail();
 jobStates.clear(); jobStateScope = '';
 jobsNotices?.replaceChildren();
 if(jobsStatus) { jobsStatus.textContent = jobsErrorMessage(code); jobsStatus.dataset.state = code; }
 if(jobsRefresh) { jobsRefresh.textContent = 'Refresh jobs'; jobsRefresh.removeAttribute('aria-busy'); }
}
function jobDate(raw: string): string {
 const date = new Date(raw);
 return Number.isNaN(date.valueOf()) ? raw : date.toLocaleString();
}
function renderJobs(jobs: MyJob[]): void {
 if(!jobsList) return;
 jobsList.replaceChildren();
 for(const job of jobs){
  const row = document.createElement('li'); row.className = 'job-entry';
  const copy = document.createElement('div'); copy.className = 'job-entry-copy';
  const kind = document.createElement('strong'); kind.textContent = job.kind;
  const id = document.createElement('span'); id.className = 'job-id'; id.textContent = job.id;
  const summary = document.createElement('span'); summary.className = 'job-summary'; summary.textContent = `${job.state} · Created ${jobDate(job.created_at)}`;
  const button = document.createElement('button'); button.type = 'button'; button.textContent = 'View details';
  button.dataset.jobId = job.id; button.setAttribute('aria-label', `View details for job ${job.id}`);
  button.setAttribute('aria-controls', 'job-detail'); button.setAttribute('aria-pressed', 'false');
  button.addEventListener('click', () => void selectJob(job.id, true));
  copy.append(kind, id, summary); row.append(copy, button); jobsList.append(row);
 }
}
async function showJobsPage(cursors: string[], moveFocus = false): Promise<void> {
 if(!jobsStatus || !jobsList || !jobsRefresh) return;
 const request = ++jobsRequest;
 jobsController?.abort(); jobsController = new AbortController();
 const controller = jobsController;
 clearJobDetail(); jobsList.replaceChildren(); jobsList.setAttribute('aria-busy', 'true');
 if(jobsPagination) jobsPagination.hidden = true;
 jobsRefresh.setAttribute('aria-busy', 'true'); jobsRefresh.textContent = 'Refresh jobs';
 jobsStatus.dataset.state = 'loading'; jobsStatus.textContent = 'Loading jobs for your current session…';
 if(moveFocus) jobsStatus.focus();
 try{
  const page = await loadMyJobsPage(cursors[cursors.length - 1], undefined, {signal: controller.signal});
  if(request !== jobsRequest || controller.signal.aborted) return;
  renderJobChanges(page.jobs);
  jobsCursors = cursors; jobsNextAfter = page.next_after;
  renderJobs(page.jobs);
  jobsStatus.dataset.state = page.jobs.length ? 'available' : 'empty';
  jobsStatus.textContent = page.jobs.length ? `Page ${cursors.length}: ${page.jobs.length} ${page.jobs.length === 1 ? 'job' : 'jobs'} returned for your current session. Select a job to read its metadata.` : 'No jobs returned for your current access.';
  if(jobsPagination && jobsPrevious && jobsNext && jobsPage) {
   jobsPrevious.disabled = cursors.length === 1; jobsNext.disabled = !page.next_after;
   jobsPage.textContent = `Page ${cursors.length}`;
   jobsPagination.hidden = cursors.length === 1 && !page.next_after;
  }
 }catch(error){
  if(request !== jobsRequest || controller.signal.aborted) return;
  const code = jobsErrorCode(error);
  if(isJobAccessFailure(code)) { clearJobsForAccessFailure(code); return; }
  jobsStatus.dataset.state = code; jobsStatus.textContent = jobsErrorMessage(code);
 }finally{
  if(request === jobsRequest) { jobsList.removeAttribute('aria-busy'); jobsRefresh.removeAttribute('aria-busy'); }
 }
}
async function selectJob(id: string, moveFocus: boolean): Promise<void> {
 if(!jobDetail || !jobDetailStatus || !jobDetailFields || !jobDetailTitle || !jobDetailRetry) return;
 const request = ++detailRequest;
 detailController?.abort(); detailController = new AbortController();
 const controller = detailController;
 selectedJob = id;
 for(const button of Array.from(jobsList?.querySelectorAll<HTMLButtonElement>('button[data-job-id]') ?? [])) button.setAttribute('aria-pressed', String(button.dataset.jobId === id));
 jobDetail.hidden = false; jobDetail.setAttribute('aria-busy', 'true');
 jobDetailFields.replaceChildren(); jobDetailRetry.hidden = true;
 jobDetailTitle.textContent = `Job ${id}`;
 jobDetailStatus.dataset.state = 'loading'; jobDetailStatus.textContent = 'Loading job metadata…';
 if(moveFocus) jobDetailTitle.focus();
 try{
  const job = await loadMyJob(id, undefined, {signal: controller.signal});
  if(request !== detailRequest || controller.signal.aborted) return;
  const values: [string,string][] = [['Job ID',job.id],['Kind',job.kind],['State',job.state],['Tenant',job.tenant_id],['Requester',job.requested_by ?? 'Not recorded'],['Created',jobDate(job.created_at)],['Updated',jobDate(job.updated_at)],['Idempotency key',job.idempotency_key]];
  for(const [label,value] of values){ const term = document.createElement('dt'); term.textContent = label; const description = document.createElement('dd'); description.textContent = value; jobDetailFields.append(term,description); }
  jobDetailStatus.dataset.state = 'available'; jobDetailStatus.textContent = 'Recorded queue metadata. A completed validation.noop job does not provision a site or change the host.';
 }catch(error){
  if(request !== detailRequest || controller.signal.aborted) return;
  const code = jobsErrorCode(error);
  if(isJobAccessFailure(code)) { clearJobsForAccessFailure(code); jobsRefresh?.focus(); return; }
  jobDetailStatus.dataset.state = code; jobDetailStatus.textContent = jobsErrorMessage(code); jobDetailRetry.hidden = false;
 }finally{ if(request === detailRequest) jobDetail.removeAttribute('aria-busy'); }
}
jobsRefresh?.addEventListener('click', () => void showJobsPage(['']));
jobsPrevious?.addEventListener('click', () => { if(jobsCursors.length > 1) void showJobsPage(jobsCursors.slice(0, -1), true); });
jobsNext?.addEventListener('click', () => { if(jobsNextAfter) void showJobsPage([...jobsCursors, jobsNextAfter], true); });
jobDetailRetry?.addEventListener('click', () => { if(selectedJob) void selectJob(selectedJob, true); });

const filesPreview = document.querySelector<HTMLElement>('#files-preview');
const filesPreviewTitle = document.querySelector<HTMLElement>('#files-preview-title');
const filesPreviewContent = document.querySelector<HTMLElement>('#files-preview-content');
const filesPreviewClose = document.querySelector<HTMLButtonElement>('#files-preview-close');
let filesPreviewRequestID = 0;
function clearFilePreview(): void {
    filesPreviewRequestID++;
    if (filesPreview) filesPreview.hidden = true;
    if (filesPreviewContent) filesPreviewContent.textContent = '';
    if (filesPreviewTitle) filesPreviewTitle.textContent = 'Text preview';
}
filesPreviewClose?.addEventListener('click', clearFilePreview);
document.addEventListener('visibilitychange', () => { if (document.hidden) clearFilePreview(); });
window.addEventListener('blur', clearFilePreview);
const filesFilter = document.querySelector<HTMLFormElement>('#files-filter');
const filesPrefixInput = document.querySelector<HTMLInputElement>('#files-prefix');
let filesPrefix = '';
let filesDisplayedPrefix: string | null = null;
const filesRefresh = document.querySelector<HTMLButtonElement>('#files-refresh');
const filesNext = document.querySelector<HTMLButtonElement>('#files-next');
const filesPrevious = document.querySelector<HTMLButtonElement>('#files-previous');
const maxFilesPages = 100;
let filesHistory: string[] = [''];
let filesDisplayedPath: string | null = null;
let filesNextAfter = '';
let filesRequestID = 0;
const filesStatus = document.querySelector<HTMLElement>('#files-status');
const filesList = document.querySelector<HTMLUListElement>('#files-list');
const filesBreadcrumbs = document.querySelector<HTMLElement>('#files-breadcrumbs');
let filesPath: string[] = [];
function formatFileSize(size: number): string {
    if (size < 1024)
        return `${size} B`;
    if (size < 1024 * 1024)
        return `${(size / 1024).toFixed(1)} KB`;
    return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}
function renderFilesBreadcrumbs() {
    if (!filesBreadcrumbs)
        return;
    filesBreadcrumbs.replaceChildren();
    const labels = ['Files', ...filesPath];
    labels.forEach((label, index) => {
        if (index > 0) {
            const separator = document.createElement('span');
            separator.textContent = '›';
            separator.setAttribute('aria-hidden', 'true');
            filesBreadcrumbs.append(separator);
        }
        const button = document.createElement('button');
        button.type = 'button';
        button.textContent = label;
        button.setAttribute('aria-current', index === labels.length - 1 ? 'location' : 'false');
        button.addEventListener('click', () => {
            filesPath = filesPath.slice(0, index);
            filesPrefix = '';
            if (filesPrefixInput) filesPrefixInput.value = '';
            void refreshFiles();
        });
        filesBreadcrumbs.append(button);
    });
}
function renderFilesEntries(entries: FileEntry[]): void {
    if (!filesList)
        return;
    filesList.replaceChildren();
    const rowPath = filesPath.join('/');
    const rowActive = () => !filesRefresh?.disabled && filesPath.join('/') === rowPath && filesDisplayedPath === rowPath;
    for (const entry of entries) {
        const item = document.createElement('li');
        item.className = 'file-entry';
        const name = document.createElement('span');
        name.className = 'file-entry-name';
        name.textContent = entry.kind === 'directory' ? `📁 ${entry.name}` : entry.name;
        const detail = document.createElement('span');
        detail.className = 'file-entry-detail';
        detail.textContent = entry.kind === 'file' ? formatFileSize(entry.size) : entry.kind === 'directory' ? 'Folder' : entry.kind === 'symlink' ? 'Link · not followed' : 'Unsupported entry';
        item.append(name, detail);
        if (entry.kind === 'directory') {
            const open = document.createElement('button');
            open.type = 'button';
            open.textContent = 'Open folder';
            open.setAttribute('aria-label', `Open folder ${entry.name}`);
            open.addEventListener('click', () => {
                if (!rowActive()) return;
                filesPath = [...filesPath, entry.name];
                filesPrefix = '';
                if (filesPrefixInput) filesPrefixInput.value = '';
                void refreshFiles();
            });
            item.append(open);
        }
        else if (entry.kind === 'file') {
            const download = document.createElement('button');
            download.type = 'button';
            download.textContent = 'Download';
            download.setAttribute('aria-label', `Download ${entry.name}`);
            download.addEventListener('click', () => { if (rowActive()) void downloadFile(entry.name, download, rowPath); });
            item.append(download);
            const preview = document.createElement('button');
            preview.type = 'button';
            preview.textContent = 'Preview text';
            preview.setAttribute('aria-label', `Preview text ${entry.name}`);
            preview.disabled = entry.size > 256 * 1024;
            preview.addEventListener('click', () => { if (rowActive()) void showFilePreview(entry.name, rowPath); });
            item.append(preview);
        }
        filesList.append(item);
    }
    if (entries.length === 0) {
        const empty = document.createElement('li');
        empty.className = 'file-entry';
        empty.textContent = 'No items on this page.';
        filesList.append(empty);
    }
}
async function refreshFiles(history: string[] = ['']) {
    if (!filesRefresh || !filesStatus || !filesList)
        return;
    clearFilePreview();
    const requestID = ++filesRequestID;
    const requestedPath = filesPath.join('/');
    const requestedPrefix = filesPrefix;
    if (filesDisplayedPath !== requestedPath || filesDisplayedPrefix !== requestedPrefix) {
        filesList.replaceChildren();
        filesDisplayedPath = null;
        filesDisplayedPrefix = null;
        filesNextAfter = '';
        filesHistory = [''];
        renderFilesBreadcrumbs();
    }
    filesRefresh.disabled = true;
    filesRefresh.setAttribute('aria-busy', 'true');
    if (filesNext) filesNext.disabled = true;
    if (filesPrevious) filesPrevious.disabled = true;
    filesStatus.dataset.state = 'loading';
    filesStatus.textContent = 'Loading files…';
    try {
        const page = await listTenantFilesPage(requestedPath, history[history.length - 1], fetch, undefined, requestedPrefix);
        if (requestID !== filesRequestID) return;
        const entries = page.entries;
        filesHistory = history;
        filesDisplayedPath = requestedPath;
        filesDisplayedPrefix = requestedPrefix;
        filesNextAfter = page.next_after;
        if (filesNext) filesNext.disabled = !filesNextAfter || filesHistory.length >= maxFilesPages;
        if (filesPrevious) filesPrevious.disabled = filesHistory.length <= 1;
        renderFilesBreadcrumbs();
        renderFilesEntries(entries);
        filesStatus.dataset.state = 'available';
        filesStatus.textContent = `Page ${filesHistory.length}: ${entries.length} ${entries.length === 1 ? 'item' : 'items'}. Folder changes may affect results; refresh to restart. Links are never followed.${filesNextAfter && filesHistory.length >= maxFilesPages ? ' Page limit reached; refresh to restart.' : ''}`;
    }
    catch (error) {
        if (requestID !== filesRequestID) return;
        renderFilesBreadcrumbs();
        if (filesDisplayedPath !== requestedPath || filesDisplayedPrefix !== requestedPrefix || (error instanceof FilesExplorerError && (error.status === 401 || error.status === 403))) {
            filesList.replaceChildren();
            filesDisplayedPath = null;
            filesDisplayedPrefix = null;
            filesHistory = [''];
            filesNextAfter = '';
        }
        if (filesPrevious) filesPrevious.disabled = filesDisplayedPath === null || filesHistory.length <= 1;
        if (filesNext) filesNext.disabled = filesDisplayedPath === null || !filesNextAfter || filesHistory.length >= maxFilesPages;
        filesStatus.dataset.state = 'unavailable';
        filesStatus.textContent = error instanceof Error ? error.message : 'Files could not be loaded. Try again.';
    }
    finally {
        if (requestID !== filesRequestID) return;
        filesRefresh.disabled = false;
        filesRefresh.removeAttribute('aria-busy');
    }
}
filesFilter?.addEventListener('submit', event => {
    event.preventDefault();
    filesPrefix = filesPrefixInput?.value ?? '';
    void refreshFiles();
});
filesNext?.addEventListener('click', () => {
    if (filesNextAfter && filesHistory.length < maxFilesPages) void refreshFiles([...filesHistory, filesNextAfter]);
});
filesPrevious?.addEventListener('click', () => {
    if (filesHistory.length > 1) void refreshFiles(filesHistory.slice(0, -1));
});
function clearFilesOnDenial(error: unknown): void {
    if (!(error instanceof FilesExplorerError) || (error.status !== 401 && error.status !== 403)) return;
    clearFilePreview();
    filesRequestID++;
    if (filesRefresh) { filesRefresh.disabled = false; filesRefresh.removeAttribute('aria-busy'); }
    filesList?.replaceChildren();
    filesDisplayedPath = null;
    filesDisplayedPrefix = null;
    filesHistory = [''];
    filesNextAfter = '';
    if (filesPrevious) filesPrevious.disabled = true;
    if (filesNext) filesNext.disabled = true;
}
async function showFilePreview(name: string, rowPath: string): Promise<void> {
    clearFilePreview();
    const requestID = filesPreviewRequestID;
    if (filesStatus) { filesStatus.dataset.state = 'loading'; filesStatus.textContent = 'Loading text preview…'; }
    try {
        const text = await previewTenantText(rowPath ? `${rowPath}/${name}` : name);
        if (requestID !== filesPreviewRequestID || document.hidden || !document.hasFocus()) return;
        if (filesPreviewTitle) filesPreviewTitle.textContent = `Text preview: ${name}`;
        if (filesPreviewContent) filesPreviewContent.textContent = text;
        if (filesPreview) filesPreview.hidden = false;
        if (filesStatus) { filesStatus.dataset.state = 'available'; filesStatus.textContent = 'Text preview loaded.'; }
    } catch (error) {
        if (requestID !== filesPreviewRequestID) return;
        clearFilePreview();
        clearFilesOnDenial(error);
        if (filesStatus) { filesStatus.dataset.state = 'unavailable'; filesStatus.textContent = error instanceof Error ? error.message : 'Text preview unavailable.'; }
    }
}
async function downloadFile(name: string, button: HTMLButtonElement, rowPath: string): Promise<void> {
    const requestID = filesRequestID;
    button.disabled = true;
    try {
        const file = await downloadTenantFile(rowPath ? `${rowPath}/${name}` : name);
        if (requestID !== filesRequestID || filesPath.join('/') !== rowPath) return;
        const objectURL = URL.createObjectURL(file.blob);
        const link = document.createElement('a');
        link.href = objectURL;
        link.download = file.name;
        link.hidden = true;
        document.body.append(link);
        link.click();
        link.remove();
        window.setTimeout(() => URL.revokeObjectURL(objectURL), 1000);
        if (filesStatus) {
            filesStatus.dataset.state = 'available';
            filesStatus.textContent = `Downloaded ${file.name}.`;
        }
    }
    catch (error) {
        if (requestID !== filesRequestID || filesPath.join('/') !== rowPath) return;
        clearFilePreview();
        clearFilesOnDenial(error);
        if (filesStatus) {
            filesStatus.dataset.state = 'unavailable';
            filesStatus.textContent = error instanceof Error ? error.message : 'The file could not be downloaded. Try again.';
        }
    }
    finally {
        button.disabled = false;
    }
}
filesRefresh?.addEventListener('click', () => void refreshFiles());
renderFilesBreadcrumbs();
