export {};
type Tile = { title: string; icon: string; subtitle: string; stage: string };
const tiles: Tile[] = [
 {title:'Websites',icon:'🌐',subtitle:'Sites & PHP',stage:'P2'},
 {title:'Domains',icon:'🪐',subtitle:'Aliases & redirects',stage:'P2'},
 {title:'Applications',icon:'🧩',subtitle:'Node.js, Python & more',stage:'P3'},
 {title:'File Explorer',icon:'📁',subtitle:'Files, uploads & editor',stage:'P2'},
 {title:'Databases',icon:'🗃️',subtitle:'MySQL & PostgreSQL',stage:'P2/P4'},
 {title:'Email',icon:'✉️',subtitle:'Mailboxes & delivery',stage:'P4'},
 {title:'SSL & HTTPS',icon:'🔒',subtitle:'Certificates & renewals',stage:'P2'},
 {title:'DNS',icon:'📡',subtitle:'Records & nameservers',stage:'P4'},
 {title:'Backups',icon:'💾',subtitle:'Back up & restore',stage:'P2'},
 {title:'Server Health',icon:'📊',subtitle:'System monitoring',stage:'P1+'},
 {title:'User Accounts',icon:'👥',subtitle:'Roles & packages',stage:'P1/P4'},
 {title:'Marketplace',icon:'🎁',subtitle:'One-click app recipes',stage:'P3+'},
];
const list = document.querySelector<HTMLElement>('#tiles');
const search = document.querySelector<HTMLInputElement>('#search');
const status = document.querySelector<HTMLElement>('#status');
const theme = document.querySelector<HTMLButtonElement>('#theme');
const intro = document.querySelector<HTMLDialogElement>('#intro');
const filterText = (text: string): void => {
 if (!list) return;
 list.replaceChildren();
 const needle=text.trim().toLowerCase();
 const shown=tiles.filter(t=>`${t.title} ${t.subtitle}`.toLowerCase().includes(needle));
 for(const t of shown){
   const card=document.createElement('article');card.className='tile';
   const ico=document.createElement('span');ico.className='tile-icon';ico.textContent=t.icon;ico.setAttribute('aria-hidden','true');
   const head=document.createElement('h3');head.textContent=t.title;
   const detail=document.createElement('p');detail.textContent=t.subtitle;
   const tag=document.createElement('span');tag.className='planned';tag.textContent=`Planned · ${t.stage}`;
   card.append(ico,head,detail,tag);list.appendChild(card);
 }
 const counter=document.querySelector<HTMLElement>('#count'); if(counter) counter.textContent=`${shown.length} modules planned`;
};
search?.addEventListener('input',()=>filterText(search.value));
filterText('');
theme?.addEventListener('click',()=>{
  const dark=document.documentElement.dataset.theme!=='dark';document.documentElement.dataset.theme=dark?'dark':'light';
  theme.setAttribute('aria-label',dark?'Switch to light mode':'Switch to dark mode');
});
document.querySelector('#tour')?.addEventListener('click',()=>intro?.showModal());
document.querySelector('#close-tour')?.addEventListener('click',()=>intro?.close());
async function ping():Promise<void>{
 if(!status)return;
 try{
  const response=await fetch('/api/v1/version',{cache:'no-store'});
  if(!response.ok)throw new Error('server unavailable');
  const data: {stage?:string} = await response.json() as {stage?:string};
  status.textContent=`Local API: ${data.stage ?? 'unknown'}`;status.classList.add('online');
 }catch{status.textContent='API offline · design preview';status.classList.remove('online');}
}
void ping();
