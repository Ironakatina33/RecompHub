/* RecompHub v2.0 - console-inspired local desktop interface */
'use strict';
const ICONS={
  grid:'<rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/>',
  star:'<path d="m12 2 3.2 6.7 7.3 1-5.3 5.1 1.3 7.2L12 18.5 5.5 22l1.3-7.2-5.3-5.1 7.3-1z"/>',
  cpu:'<rect x="5" y="5" width="14" height="14" rx="2"/><path d="M9 1v4M15 1v4M9 19v4M15 19v4M19 9h4M19 15h4M1 9h4M1 15h4"/>',
  monitor:'<rect x="2" y="3" width="20" height="15" rx="2"/><path d="M8 22h8M12 18v4"/>',
  rocket:'<path d="m4 14 6 6 3-4 6-6c2-2 2-7 2-7s-5 0-7 2l-6 6z"/><path d="m6 17-2 2-1 3 3-1 2-2M9 8l7 7"/><circle cx="16" cy="8" r="2"/>',
  disc:'<circle cx="12" cy="12" r="9"/><circle cx="12" cy="12" r="2"/><path d="M17 5a9 9 0 0 1 2 3"/>',
  settings:'<circle cx="12" cy="12" r="3"/><path d="m19 13 2 1-2 4-2-1-2 2v2h-5v-2l-2-2-2 1-2-4 2-1v-2L4 10l2-4 2 1 2-2V3h5v2l2 2 2-1 2 4-2 1z"/>',
  refresh:'<path d="M20 11A8 8 0 0 0 6 5L4 7M4 3v4h4M4 13a8 8 0 0 0 14 6l2-2m0 4v-4h-4"/>',
  search:'<circle cx="11" cy="11" r="7"/><path d="m16 16 5 5"/>',
  plus:'<path d="M12 5v14M5 12h14"/>',
  'folder-search':'<path d="M3 7a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><circle cx="12" cy="13" r="3"/><path d="m14.5 15.5 2 2"/>',
  play:'<path d="m8 4 12 8-12 8z"/>',
  more:'<circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/>',
  'chevron-down':'<path d="m6 9 6 6 6-6"/>',
  x:'<path d="M18 6 6 18M6 6l12 12"/>',
  check:'<path d="m5 12 4 4 10-10"/>',
  'check-circle':'<circle cx="12" cy="12" r="9"/><path d="m8 12 3 3 5-6"/>',
  image:'<rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="m21 16-6-6L4 21"/>',
  gamepad:'<path d="M6 9h12a4 4 0 0 1 4 3l1 6a2 2 0 0 1-3 2l-3-2H7l-3 2a2 2 0 0 1-3-2l1-6a4 4 0 0 1 4-3z"/><path d="M7 11v6M4 14h6M16 13h.01M19 16h.01"/>',
  palette:'<path d="M12 2a10 10 0 0 0 0 20h1a2 2 0 0 0 2-2c0-2-1-3 1-3h2a4 4 0 0 0 4-4c0-6-4-11-10-11z"/><circle cx="6.5" cy="12.5" r="1"/><circle cx="9" cy="7" r="1"/><circle cx="15" cy="7" r="1"/><circle cx="18" cy="12" r="1"/>',
  'cloud-download':'<path d="M20 17a5 5 0 0 0-2-9 6 6 0 0 0-11 2 4 4 0 0 0-1 7"/><path d="M12 12v9m-4-4 4 4 4-4"/>',
  'hard-drive':'<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M3 13h18M8 17h.01M12 17h.01"/>',
  folder:'<path d="M3 6a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>',
  trash:'<path d="M4 7h16M10 4h4M6 7l1 14h10l1-14M10 11v6M14 11v6"/>',
  clock:'<circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/>',
  external:'<path d="M13 5h6v6M19 5l-9 9"/><path d="M19 13v6H5V5h6"/>'
};
function icon(id){return `<svg aria-hidden="true" viewBox="0 0 24 24">${ICONS[id]||ICONS.disc}</svg>`}
function injectIcons(){document.querySelectorAll('[data-icon]').forEach(el=>{el.innerHTML=icon(el.dataset.icon)})}
function e(s){return String(s??'').replace(/[&<>"']/g,ch=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]))}
const THEMES=[
  {id:'neon',name:'Neon Arcade',caption:'Bleu nuit · Néons et transparences',symbol:'✦'},
  {id:'ps3',name:'XMB Legacy',caption:'PS3 · Vagues et navigation fluide',symbol:'✧'},
  {id:'ps4',name:'Blue Horizon',caption:'PS4 · Grandes tuiles bleues',symbol:'◈'},
  {id:'wii',name:'Channel Studio',caption:'Wii · Chaînes et ambiance lumineuse',symbol:'◉'},
  {id:'nintendo',name:'Joy Home',caption:'Nintendo · Clair et coloré',symbol:'●'}
];
const NAV=[['all','Tout voir','grid'],['fav','Favoris','star'],['Recompilation','Recompilations','cpu'],['Port PC','Ports PC','monitor'],['Launcher','Launchers','rocket'],['Autre','Autres','disc']];
const COLORS=[['#1e3766','#6153b2'],['#255b89','#40a9bb'],['#8b405f','#d96952'],['#0e5969','#2766ad'],['#5b416b','#bc618c'],['#48785d','#94bf63'],['#7b4a38','#d1945b'],['#26375a','#3d9fb2']];
const DEMO=[
 {id:'demo-a',name:'Ocarina of Time',category:'Recompilation',favorite:true,playCount:20,added:'2026-09-20',lastPlayed:'2026-10-26',cover:'',path:'C:\\Games\\Ocarina\\game.exe'},
 {id:'demo-b',name:'Majora’s Mask',category:'Recompilation',favorite:false,playCount:6,added:'2026-10-02',lastPlayed:'2026-10-18',cover:'',path:'C:\\Games\\Majora\\game.exe'},
 {id:'demo-c',name:'Sonic Mania',category:'Port PC',favorite:true,playCount:9,added:'2026-09-29',lastPlayed:'2026-10-22',cover:'',path:'C:\\Games\\Sonic\\game.exe'},
 {id:'demo-d',name:'Ship of Harkinian',category:'Launcher',favorite:false,playCount:5,added:'2026-10-03',lastPlayed:'2026-10-09',cover:'',path:'C:\\Games\\Ship\\game.exe'},
 {id:'demo-e',name:'Mario 64',category:'Port PC',favorite:false,playCount:2,added:'2026-10-10',lastPlayed:'2026-10-12',cover:'',path:'C:\\Games\\Mario\\game.exe'},
 {id:'demo-f',name:'Star Fox 64',category:'Recompilation',favorite:false,playCount:1,added:'2026-10-12',lastPlayed:'2026-10-15',cover:'',path:'C:\\Games\\StarFox\\game.exe'},
 {id:'demo-g',name:'OpenGOAL',category:'Launcher',favorite:false,playCount:3,added:'2026-10-19',lastPlayed:'2026-10-23',cover:'',path:'C:\\Games\\OpenGOAL\\game.exe'},
 {id:'demo-h',name:'Banjo Recompiled',category:'Recompilation',favorite:false,playCount:1,added:'2026-10-20',lastPlayed:'2026-10-24',cover:'',path:'C:\\Games\\Banjo\\game.exe'},
];
const DEMO_MODE=new URLSearchParams(location.search).has('demo');
const data={games:[],theme:'neon',filter:'all',q:'',sort:'recent',selected:'',version:'2.0.0',repo:'Ironakatina33/RecompHub',autoCheck:true,update:null,loading:false};
let toastHandle;
async function api(path,body){
 if(DEMO_MODE){return {ok:true}}
 const options={method:body===undefined?'GET':'POST',headers:{'X-RecompHub-Token':window.RECOMPHUB_TOKEN||''}};
 if(body!==undefined){options.headers['Content-Type']='application/json';options.body=JSON.stringify(body)}
 let response;try{response=await fetch('/api/'+path,options)}catch(err){throw Error('Connexion à RecompHub perdue. Relance l’application.')}
 let result;try{result=await response.json()}catch{throw Error('Réponse incorrecte du programme')}
 if(!response.ok||result.ok===false)throw Error(result.error||'Une erreur est survenue');
 return result;
}
async function loadLibrary(){
 if(DEMO_MODE){data.games=structuredClone(DEMO);data.theme='neon';data.version='2.0.0';render();return}
 try{const result=await api('library');data.games=result.library.games||[];data.theme=result.library.settings?.Theme||'neon';data.autoCheck=result.library.settings?.AutoCheck!==false;data.version=result.version||'2.0.0';data.repo=result.repo||data.repo;render();if(data.autoCheck)setTimeout(()=>checkUpdate(false),1800)}catch(err){toast(err.message,true);render()}
}
function toast(message,error=false){const el=document.getElementById('toast');el.textContent=(error?'⚠  ':'✓  ')+message;el.classList.add('visible');clearTimeout(toastHandle);toastHandle=setTimeout(()=>el.classList.remove('visible'),4400)}
function render(){
 if(!THEMES.some(t=>t.id===data.theme))data.theme='neon';
 document.body.dataset.theme=data.theme;
 const theme=THEMES.find(t=>t.id===data.theme);
 document.getElementById('currentThemeName').textContent=theme.name;
 document.getElementById('settingsThemeName').textContent=theme.name;
 document.getElementById('miniThemeSymbol').textContent=theme.symbol;
 document.getElementById('settingsVersion').textContent=data.version;
 document.getElementById('buildLabel').textContent='v'+data.version;
 document.getElementById('sidebarCount').textContent=`${data.games.length} jeu${data.games.length!==1?'x':''}`;
 document.getElementById('autoUpdateToggle').checked=data.autoCheck;
 const filtered=filteredGames();
 document.getElementById('countBubble').textContent=String(filtered.length).padStart(2,'0');
 const selectedNav=NAV.find(n=>n[0]===data.filter)||NAV[0];
 document.getElementById('categoryLabel').textContent=selectedNav[1];
 document.getElementById('sectionTitle').innerHTML=`${data.filter==='all'?'Ma bibliothèque':e(selectedNav[1])} <span class="count-bubble" id="countBubble">${String(filtered.length).padStart(2,'0')}</span>`;
 renderNav();renderCards(filtered);renderHero();renderThemes();injectIcons();
}
function filteredGames(){
 const term=data.q.trim().toLowerCase();let games=data.games.filter(g=>(data.filter==='all'||(data.filter==='fav'?g.favorite:g.category===data.filter))&&(!term||[g.name,g.category,g.path].some(s=>String(s||'').toLowerCase().includes(term))));
 switch(data.sort){case 'az':games.sort((a,b)=>a.name.localeCompare(b.name,'fr'));break;case 'played':games.sort((a,b)=>String(b.lastPlayed||'').localeCompare(String(a.lastPlayed||'')));break;case 'most':games.sort((a,b)=>(b.playCount||0)-(a.playCount||0));break;default:games.sort((a,b)=>String(b.added||'').localeCompare(String(a.added||'')))}
 return games;
}
function renderNav(){
 const counts={all:data.games.length,fav:data.games.filter(g=>g.favorite).length};
 for(const g of data.games)counts[g.category]=(counts[g.category]||0)+1;
 document.getElementById('navStack').innerHTML=NAV.map(([id,label,ic])=>`<button title="${e(label)}" class="nav-item ${data.filter===id?'active':''}" data-action="filter" data-id="${e(id)}"><span class="nav-ic">${icon(ic)}</span><span class="nav-title">${e(label)}</span><span class="nav-count">${counts[id]||0}</span></button>`).join('');
}
function monogram(name){const words=String(name).split(/\s+/).filter(Boolean);return (words.length>1?words.slice(0,2).map(s=>s[0]).join(''):name.slice(0,2)).toUpperCase()}
function renderCards(games){
 document.getElementById('emptyState').hidden=games.length!==0;
 if(!games.length){document.getElementById('emptyTitle').textContent=data.games.length?'Aucun jeu trouvé.':'Une collection à créer.';document.getElementById('emptyDesc').textContent=data.games.length?'Essaie une autre recherche ou catégorie.':'Ajoute ton premier jeu ou scanne un dossier pour commencer.'}
 document.getElementById('gamesGrid').innerHTML=games.map((g,index)=>{
   const palette=COLORS[Math.abs([...g.name].reduce((s,c)=>s+c.charCodeAt(0),0))%COLORS.length];
   const cat=g.category==='Recompilation'?'RECOMP':g.category==='Port PC'?'PORT PC':String(g.category||'AUTRE').toUpperCase();
   const cover=g.cover?`<img class="cover-image" src="/api/cover?id=${encodeURIComponent(g.id)}&token=${encodeURIComponent(window.RECOMPHUB_TOKEN||'')}" alt="" onerror="this.style.display='none'"/>`:'';
   return `<article class="game-card" style="--cover-one:${palette[0]};--cover-two:${palette[1]};animation-delay:${Math.min(index*35,250)}ms"><div class="cover-art">${cover}<div class="cover-monogram">${e(monogram(g.name))}</div><span class="game-type">${e(cat)}</span><button class="fav-button ${g.favorite?'saved':''}" data-action="favorite" data-id="${e(g.id)}" title="Favoris" aria-label="${g.favorite?'Retirer des':'Ajouter aux'} favoris">${icon('star')}</button></div><div class="game-meta"><h3 title="${e(g.name)}">${e(g.name)}</h3><p>${(g.playCount||0)>0?`${g.playCount} session${g.playCount>1?'s':''}`:'Prêt à jouer'} · ${e(g.category||'Port PC')}</p></div><div class="game-bottom"><button class="game-launch" data-action="play" data-id="${e(g.id)}">${icon('play')} JOUER</button><button class="game-edit" data-action="edit" data-id="${e(g.id)}" title="Modifier le jeu" aria-label="Modifier">${icon('more')}</button></div></article>`;
 }).join('');
}
function renderHero(){
 const recent=[...data.games].filter(g=>g.lastPlayed).sort((a,b)=>String(b.lastPlayed).localeCompare(String(a.lastPlayed)))[0];
 const primary=document.getElementById('heroPrimary');
 if(recent){
   document.getElementById('welcomeEyebrow').textContent='REPRENDRE TA PARTIE';
   document.getElementById('welcomeTitle').innerHTML=`Ton prochain<br/><em>voyage t’attend.</em>`;
   document.getElementById('welcomeDesc').textContent=`Tu jouais à ${recent.name}. Prêt à replonger ?`;
   primary.dataset.action='play';primary.dataset.id=recent.id;document.getElementById('heroPrimaryText').textContent='Reprendre '+recent.name;
 }else{
   document.getElementById('welcomeEyebrow').textContent='TON ESPACE DE JEU';
   document.getElementById('welcomeTitle').innerHTML='Tous tes jeux.<br/><em>Un seul endroit.</em>';
   document.getElementById('welcomeDesc').textContent='Recompilations, ports PC et launchers réunis dans une bibliothèque faite pour toi.';
   primary.dataset.action='add';delete primary.dataset.id;document.getElementById('heroPrimaryText').textContent='Ajouter un jeu';
 }
}
function renderThemes(){
 document.getElementById('themeGallery').innerHTML=THEMES.map(t=>`<button class="theme-option ${data.theme===t.id?'selected':''}" data-action="select-theme" data-id="${t.id}" aria-pressed="${data.theme===t.id}"><div class="theme-preview ${t.id}"><div class="preview-top"></div><div class="preview-tiles"><i></i><i></i><i></i><i></i></div></div>${data.theme===t.id?`<span class="chosen-icon">${icon('check')}</span>`:''}<strong>${e(t.name)}</strong><small>${e(t.caption)}</small></button>`).join('');
}
function showModal(name){closeModal();document.getElementById(name+'Overlay').hidden=false;document.body.style.overflow='hidden';if(name==='theme')renderThemes();injectIcons()}
function closeModal(){document.querySelectorAll('.overlay').forEach(el=>el.hidden=true);document.body.style.overflow=''}
async function setTheme(id){if(!THEMES.some(t=>t.id===id))return;const previous=data.theme;data.theme=id;render();try{if(!DEMO_MODE)await api('settings',{theme:id});toast(`Thème « ${THEMES.find(t=>t.id===id).name} » activé !`)}catch(err){data.theme=previous;render();toast(err.message,true)}}
async function addGame(){if(DEMO_MODE){toast('Aperçu uniquement : utilise le .exe pour ajouter tes jeux.');return}toast('Choisis un programme dans la fenêtre Windows...');try{const res=await api('add',{});if(res.cancelled)return;await refreshLibrary();toast(res.added?'Jeu ajouté à ta bibliothèque !':'Ce jeu est déjà présent.')}catch(err){toast(err.message,true)}}
async function scanGames(){if(DEMO_MODE){toast('Aperçu uniquement : utilise le .exe pour scanner tes dossiers.');return}toast('Sélectionne un dossier dans la fenêtre Windows...');try{const res=await api('scan',{});if(res.cancelled)return;await refreshLibrary();toast(`${res.added} jeu(x) ajouté(s) · ${res.visited} dossiers parcourus`)}catch(err){toast(err.message,true)}}
async function refreshLibrary(){if(DEMO_MODE){render();return}const result=await api('library');data.games=result.library.games||[];data.theme=result.library.settings?.Theme||'neon';data.autoCheck=result.library.settings?.AutoCheck!==false;render()}
async function favorite(id){if(DEMO_MODE){const g=data.games.find(g=>g.id===id);if(g)g.favorite=!g.favorite;render();return}try{await api('favorite',{id});await refreshLibrary()}catch(err){toast(err.message,true)}}
async function play(id){if(DEMO_MODE){toast('Aperçu : le jeu se lancerait ici !');return}try{await api('play',{id});await refreshLibrary();toast('Bon jeu ! 🎮')}catch(err){toast(err.message,true)}}
function edit(id){const g=data.games.find(g=>g.id===id);if(!g)return;data.selected=id;const form=document.getElementById('editForm');for(const field of ['Name','Category','Args','Notes'])form.elements[field].value=g[field.toLowerCase()]||'';document.getElementById('editPath').textContent=g.path;showModal('edit')}
async function saveEdit(ev){ev.preventDefault();const id=data.selected;const form=ev.currentTarget;const values={id};for(const field of ['Name','Category','Args','Notes'])values[field]=form.elements[field].value;if(DEMO_MODE){const g=data.games.find(g=>g.id===id);if(g){g.name=values.Name;g.category=values.Category;g.args=values.Args;g.notes=values.Notes}closeModal();render();toast('Modifications enregistrées (aperçu)');return}try{await api('edit',values);closeModal();await refreshLibrary();toast('Jeu modifié')}catch(err){toast(err.message,true)}}
async function chooseCover(){if(DEMO_MODE){toast('Couvertures personnalisées disponibles dans le .exe');return}try{const res=await api('cover/pick',{id:data.selected});if(!res.cancelled){await refreshLibrary();toast('Couverture mise à jour !')}}catch(err){toast(err.message,true)}}
async function removeGame(){if(!data.selected)return;const g=data.games.find(g=>g.id===data.selected);if(!g||!confirm(`Retirer « ${g.name} » de la bibliothèque ?\nLe fichier du jeu restera sur ton PC.`))return;if(DEMO_MODE){data.games=data.games.filter(x=>x.id!==g.id);closeModal();render();return}try{await api('remove',{id:g.id});closeModal();await refreshLibrary();toast('Jeu retiré (fichiers conservés)')}catch(err){toast(err.message,true)}}
async function reveal(){if(DEMO_MODE){toast('Option disponible dans le .exe');return}try{await api('reveal',{id:data.selected})}catch(err){toast(err.message,true)}}
async function checkUpdate(manual){if(DEMO_MODE){if(manual)toast('Mode aperçu : mises à jour disponibles dans le .exe');return}const status=document.getElementById('updateStatusText');status.textContent='Recherche en cours...';try{const result=await api('update/check');if(result.available){data.update=result.manifest;status.textContent=`Version ${result.manifest.version} disponible`;document.getElementById('updateIcon').classList.add('update-ready');if(confirm(`RecompHub ${result.manifest.version} est disponible.\n\nTélécharger et installer cette mise à jour maintenant ?\nTa bibliothèque sera conservée.`)){toast('Téléchargement et vérification en cours...');try{await api('update/apply',data.update);toast('Mise à jour installée : redémarrage...')}catch(err){toast(err.message,true)}}}else{status.textContent=`À jour (v${data.version})`;if(manual)toast('Tu utilises déjà la dernière version !')}}catch(err){status.textContent='Vérification indisponible';if(manual)toast(err.message,true)}}
async function changeAutoCheck(checked){data.autoCheck=checked;if(DEMO_MODE)return;try{await api('settings',{autoCheck:checked});toast(checked?'Vérification automatique activée':'Vérification automatique désactivée')}catch(err){toast(err.message,true)}}
function startClock(){const tick=()=>{const now=new Date();const value=now.toLocaleTimeString('fr-FR',{hour:'2-digit',minute:'2-digit'});document.getElementById('clock').textContent=value;document.getElementById('dockClock').textContent=value};tick();setInterval(tick,15000)}
function handleClick(ev){const el=ev.target.closest('[data-action]');if(!el)return;const action=el.dataset.action;const id=el.dataset.id;switch(action){case 'filter':data.filter=id;render();break;case 'themes':showModal('theme');break;case 'settings':showModal('settings');break;case 'close-modal':closeModal();break;case 'select-theme':setTheme(id);break;case 'add':addGame();break;case 'scan':scanGames();break;case 'favorite':favorite(id);break;case 'play':play(id);break;case 'edit':edit(id);break;case 'choose-cover':chooseCover();break;case 'remove':removeGame();break;case 'reveal':reveal();break;case 'updates':checkUpdate(true);break;}}
function init(){injectIcons();startClock();document.addEventListener('click',handleClick);document.getElementById('searchInput').addEventListener('input',ev=>{data.q=ev.target.value;render()});document.getElementById('sortSelect').addEventListener('change',ev=>{data.sort=ev.target.value;render()});document.getElementById('editForm').addEventListener('submit',saveEdit);document.getElementById('autoUpdateToggle').addEventListener('change',ev=>changeAutoCheck(ev.target.checked));document.querySelectorAll('.overlay').forEach(el=>el.addEventListener('click',ev=>{if(ev.target===el)closeModal()}));document.addEventListener('keydown',ev=>{if(ev.key==='Escape'){closeModal()};if((ev.ctrlKey||ev.metaKey)&&ev.key.toLowerCase()==='k'){ev.preventDefault();closeModal();document.getElementById('searchInput').focus()}});loadLibrary()}
document.addEventListener('DOMContentLoaded',init);
